package gen

import (
	"strings"
	"testing"

	"github.com/spriteCloud/quail-core/composer"
)

// TestStepsBDDTemplate_ExtractsAllRegisteredPatterns is the gen↔composer
// wiring guard. The embedded pw_steps_bdd.tmpl must yield a non-empty
// pattern slice via composer.ExtractStepPatterns — otherwise the
// composer would silently fall back to its hardcoded legacy list,
// which is exactly the drift this exporter exists to prevent.
func TestStepsBDDTemplate_ExtractsAllRegisteredPatterns(t *testing.T) {
	body := StepsBDDTemplate()
	if len(body) == 0 {
		t.Fatal("StepsBDDTemplate() returned empty bytes")
	}
	patterns := composer.ExtractStepPatterns(body)
	if len(patterns) < 30 {
		t.Fatalf("expected ≥30 patterns extracted; got %d", len(patterns))
	}
	// Spot-check the v0.3.0 widening pattern made it through.
	var seenV03 bool
	for _, p := range patterns {
		if strings.Contains(p.Raw, `the URL has query parameter`) {
			seenV03 = true
			break
		}
	}
	if !seenV03 {
		t.Error("v0.3.0 widening pattern not found in extracted set")
	}
	// Every pattern should carry a keyword the composer prompt can use.
	for _, p := range patterns {
		if p.Keyword != "Given" && p.Keyword != "When" && p.Keyword != "Then" {
			t.Errorf("pattern keyword %q is not Given/When/Then: raw=%s", p.Keyword, p.Raw)
		}
	}
}
