// Package browser hosts the sidecar Playwright probe script and the
// helper that writes it to a tempfile so `node` can exec it. The script
// is embedded via go:embed so the binary stays a single artifact.
package browser

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed probe.mjs
var probeScript []byte

// WriteScript materialises probe.mjs into a subdirectory of baseDir and
// returns the path. Callers pass the shared Playwright runner dir
// (see RunnerDir / EnsureRunner) so Node's ESM resolver walks UP from
// the temp subdir and finds @playwright/test in
// <runner>/node_modules — independent of where the probed project
// lives on disk.
//
// Caller is responsible for cleanup via the returned cleanup func.
func WriteScript(baseDir string) (path string, cleanup func(), err error) {
	if baseDir == "" {
		baseDir = "."
	}
	dir, err := os.MkdirTemp(baseDir, ".quail-browser-probe-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("browser: tempdir: %w", err)
	}
	p := filepath.Join(dir, "probe.mjs")
	if err := os.WriteFile(p, probeScript, 0o600); err != nil {
		os.RemoveAll(dir)
		return "", func() {}, fmt.Errorf("browser: write script: %w", err)
	}
	return p, func() { os.RemoveAll(dir) }, nil
}
