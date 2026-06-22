package probe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// stateFilePath is the marker file we drop after a successful probe so
// subsequent runs (against the same target URLs, within stateTTL) can
// skip the probe entirely. Kept under tests/e2e/ so it travels with
// the suite in the bot PR; the bot writer overwrites it on the next
// successful probe.
const stateRelPath = "tests/e2e/.quail-probe-state.json"

// stateTTL is how long a probe-state marker stays valid. 24h is
// long enough for a working day of follow-up PRs to skip probing,
// short enough that a stale state self-heals overnight.
const stateTTL = 24 * time.Hour

// State is the on-disk payload. Renderable to JSON so a human can eyeball
// it (it sits next to test specs).
type State struct {
	TargetURLs []string  `json:"target_urls"`
	EmittedAt  time.Time `json:"emitted_at"`
	QuailVer   string    `json:"quail_version,omitempty"`
}

// ReadState returns the state at workdir, or (zero, false) if missing
// / unparseable. Never errors — a corrupt state file is treated the
// same as a missing one: the probe re-runs.
func ReadState(workdir string) (State, bool) {
	raw, err := os.ReadFile(filepath.Join(workdir, stateRelPath))
	if err != nil {
		return State{}, false
	}
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return State{}, false
	}
	return s, true
}

// WriteState drops the marker. Failure is logged at the call site but
// not fatal — the probe ran fine; the cache miss next time isn't worth
// failing for.
func WriteState(workdir string, urls []string, quailVer string) error {
	dest := filepath.Join(workdir, stateRelPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	s := State{
		TargetURLs: normalizeURLs(urls),
		EmittedAt:  time.Now().UTC(),
		QuailVer:   quailVer,
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dest, raw, 0o644)
}

// SuiteAlreadyCovers reports true when a fresh-enough state marker
// exists AND its recorded target URLs match the requested set. Order
// and duplicates are normalised so cosmetic differences don't force
// re-probes.
func SuiteAlreadyCovers(workdir string, urls []string, now time.Time) bool {
	s, ok := ReadState(workdir)
	if !ok {
		return false
	}
	if now.Sub(s.EmittedAt) > stateTTL {
		return false
	}
	want := normalizeURLs(urls)
	have := normalizeURLs(s.TargetURLs)
	if len(want) != len(have) {
		return false
	}
	for i := range want {
		if want[i] != have[i] {
			return false
		}
	}
	return true
}

func normalizeURLs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, u := range in {
		if u == "" {
			continue
		}
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}
