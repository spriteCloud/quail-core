// Package integration loads quail.yml — the optional consumer-side
// config that declares external systems (DBs / brokers / caches /
// storage / search / auth) the integration-test generator should emit
// Testcontainers + round-trip tests for.
//
// The file lives at the repo root. Missing file is not an error — the
// integration test family stays unemitted in that case.
package integration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the schema of quail.yml.
type Config struct {
	Databases []Database `yaml:"databases"`
	Brokers   []Broker   `yaml:"brokers"`
	Caches    []Cache    `yaml:"caches"`
	Storage   []Storage  `yaml:"storage"`
	Search    []Search   `yaml:"search"`
	Auth      []Auth     `yaml:"auth"`
}

type Database struct {
	Name       string `yaml:"name"`
	Driver     string `yaml:"driver"`     // postgres | mysql | mariadb | sqlite | mongodb | …
	Image      string `yaml:"image"`      // Docker image for Testcontainers
	Migrations string `yaml:"migrations"` // optional path to migration files
}

type Broker struct {
	Kind   string   `yaml:"kind"`   // kafka | rabbitmq | nats | redis-streams
	Image  string   `yaml:"image"`
	Topics []string `yaml:"topics"`
}

type Cache struct {
	Kind  string `yaml:"kind"`  // redis | memcached
	Image string `yaml:"image"`
}

type Storage struct {
	Kind   string `yaml:"kind"`   // s3 | minio | gcs | azure-blob
	Bucket string `yaml:"bucket"`
}

type Search struct {
	Kind string `yaml:"kind"` // elasticsearch | opensearch | meilisearch
}

type Auth struct {
	Provider string `yaml:"provider"` // oidc | oauth2 | jwt
	Issuer   string `yaml:"issuer"`
}

// Load reads quail.yml from the given directory. Returns
// (nil, nil) when the file is absent; (nil, err) on parse errors.
func Load(workDir string) (*Config, error) {
	for _, name := range []string{"quail.yml", "quail.yaml"} {
		path := filepath.Join(workDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("integration: read %s: %w", path, err)
		}
		var cfg Config
		if err := yaml.Unmarshal(body, &cfg); err != nil {
			return nil, fmt.Errorf("integration: parse %s: %w", path, err)
		}
		return &cfg, nil
	}
	return nil, nil
}

// IsEmpty reports whether the config declares no integrations at all.
func (c *Config) IsEmpty() bool {
	if c == nil {
		return true
	}
	return len(c.Databases) == 0 &&
		len(c.Brokers) == 0 &&
		len(c.Caches) == 0 &&
		len(c.Storage) == 0 &&
		len(c.Search) == 0 &&
		len(c.Auth) == 0
}
