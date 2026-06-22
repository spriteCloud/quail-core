package composer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Cache is a file-backed memo of LLM responses, keyed by the
// (model, journey URL, journey kind, page-state hash) tuple. Re-running
// the probe against an unchanged site returns the previously-validated
// scenarios instantly — saves ~7s per journey on a typical run.
//
// Cache files are versioned: a schema bump (or a major composer
// change) invalidates the cache by changing the version byte at the
// head of each entry.
type Cache struct {
	// Dir is the on-disk directory holding one JSON file per cache
	// entry. Created on first write. Empty Dir disables the cache.
	Dir string
}

const cacheSchemaVersion = "v1"

// CacheKey returns the deterministic fingerprint of a (journey,
// model) pair. The fingerprint covers the data fields the prompt
// actually uses, NOT the journey's incidental fields — so a change
// to e.g. PageURL invalidates, but a fresh probe with identical
// page state hits the cache.
func CacheKey(model string, j Journey) string {
	h := sha256.New()
	h.Write([]byte(cacheSchemaVersion))
	h.Write([]byte{0})
	h.Write([]byte(model))
	h.Write([]byte{0})
	h.Write([]byte(j.URL))
	h.Write([]byte{0})
	h.Write([]byte(j.Kind))
	h.Write([]byte{0})
	h.Write([]byte(j.Title))
	h.Write([]byte{0})
	h.Write([]byte(j.H1))
	h.Write([]byte{0})
	for _, l := range j.Links {
		h.Write([]byte(l))
		h.Write([]byte{0})
	}
	h.Write([]byte{0})
	for _, p := range j.Pages {
		h.Write([]byte(p.Href))
		h.Write([]byte{0})
		h.Write([]byte(p.Title))
		h.Write([]byte{0})
		h.Write([]byte(p.H1))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a previously-cached scenario set. Returns (nil, false)
// when the entry doesn't exist or the cache is disabled. Errors
// reading a malformed entry surface as a cache miss (silent), so a
// corrupt cache never breaks generation.
func (c Cache) Get(key string) ([]ExtraScenario, bool) {
	if c.Dir == "" || key == "" {
		return nil, false
	}
	body, err := os.ReadFile(filepath.Join(c.Dir, key+".json"))
	if err != nil {
		return nil, false
	}
	var out []ExtraScenario
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, false
	}
	return out, true
}

// Put writes scenarios to the cache. Creates the cache directory on
// first write. Errors are returned but not fatal — the caller is
// expected to log + carry on (a failed cache write is not worth
// failing the run for).
func (c Cache) Put(key string, scenarios []ExtraScenario) error {
	if c.Dir == "" {
		return errors.New("composer: cache disabled (empty Dir)")
	}
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}
	body, err := json.Marshal(scenarios)
	if err != nil {
		return err
	}
	tmp := filepath.Join(c.Dir, key+".json.tmp")
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(c.Dir, key+".json"))
}

// ResolveCacheDir picks the cache directory. Precedence:
//   1. Explicit dir argument (e.g. from --llm-cache flag)
//   2. QUAIL_LLM_CACHE env var
//   3. ".quail-cache" under the work dir (off by default — empty
//      string returned when no signal opts in)
//
// Empty string return DISABLES the cache. This keeps the cache
// strictly opt-in to avoid surprise on-disk state.
func ResolveCacheDir(explicit, workDir string) string {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return explicit
	}
	if env := strings.TrimSpace(os.Getenv("QUAIL_LLM_CACHE")); env != "" {
		if env == "auto" {
			return filepath.Join(workDir, ".quail-cache")
		}
		return env
	}
	return ""
}
