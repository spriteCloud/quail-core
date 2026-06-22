package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type HealMode string

const (
	HealOnFailure HealMode = "on-failure"
	HealProactive HealMode = "proactive"
	HealOff       HealMode = "off"
)

type Config struct {
	GitHubToken      string
	Repo             string
	PRNumber         int
	OpenAIBaseURL    string
	OpenAIAPIKey     string
	Model            string
	LLMTimeout       time.Duration
	LLMTokenCap      int
	HealMode         HealMode
	AllowDiffToLLM   bool
	BranchPrefix     string
	DryRun           bool
	PlaywrightReport string
	WorkDir          string
}

func FromEnv() Config {
	c := Config{
		GitHubToken:      first(os.Getenv("QUAIL_GITHUB_TOKEN"), os.Getenv("GITHUB_TOKEN")),
		Repo:             os.Getenv("GITHUB_REPOSITORY"),
		OpenAIBaseURL:    firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), "https://api.openai.com/v1"),
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		Model:            firstNonEmpty(os.Getenv("QUAIL_MODEL"), "gpt-4o-mini"),
		// v0.48 — 20s was tight against a local-Ollama-on-DGX setup
		// where responses arrived in 20-25s; we'd amputate ~50% of
		// real-world calls. 60s gives slower hardware breathing room
		// without unduly extending fast-LLM runs (those return well
		// inside it anyway). Operators can still override via the
		// env var.
		LLMTimeout:       envDuration("QUAIL_LLM_TIMEOUT", 60*time.Second),
		LLMTokenCap:      envInt("QUAIL_LLM_TOKEN_CAP", 600),
		HealMode:         HealMode(firstNonEmpty(os.Getenv("QUAIL_HEAL_MODE"), string(HealOnFailure))),
		AllowDiffToLLM:   os.Getenv("QUAIL_ALLOW_DIFF_TO_LLM") == "1",
		BranchPrefix:     firstNonEmpty(os.Getenv("QUAIL_BRANCH_PREFIX"), "quail"),
		PlaywrightReport: os.Getenv("QUAIL_PLAYWRIGHT_REPORT"),
		WorkDir:          firstNonEmpty(os.Getenv("QUAIL_WORKDIR"), "."),
	}
	if n := os.Getenv("QUAIL_PR"); n != "" {
		if v, err := strconv.Atoi(n); err == nil {
			c.PRNumber = v
		}
	}
	return c
}

func (c Config) Validate() error {
	switch c.HealMode {
	case HealOnFailure, HealProactive, HealOff:
	default:
		return fmt.Errorf("invalid QUAIL_HEAL_MODE %q; want on-failure|proactive|off", c.HealMode)
	}
	return nil
}

func (c Config) SplitRepo() (owner, name string, ok bool) {
	parts := strings.SplitN(c.Repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
