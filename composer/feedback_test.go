package composer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFeedback_ParsesLedger(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tests/e2e/docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `# Bug discovery ledger
| Spec | Test | Symptom | First seen | Last seen | Severity | Status |
|---|---|---|---|---|---|---|
| ` + "`tests/e2e/x.spec.ts`" + ` | submit blocked on empty input | 5xx | 2026-06-15 | 2026-06-17 | high | open |
| ` + "`tests/e2e/y.spec.ts`" + ` | email field accepts whitespace | 200 + bad data | 2026-06-16 | 2026-06-17 | medium | open |
`
	if err := os.WriteFile(filepath.Join(dir, "tests/e2e/docs/findings.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	fb := LoadFeedback(dir)
	if len(fb.FailedTitles) != 2 {
		t.Fatalf("expected 2 failed titles; got %d (%+v)", len(fb.FailedTitles), fb.FailedTitles)
	}
	if !strings.Contains(fb.FailedTitles[0], "submit blocked") {
		t.Errorf("unexpected first title: %q", fb.FailedTitles[0])
	}
}

func TestLoadFeedback_MissingLedgerReturnsEmpty(t *testing.T) {
	fb := LoadFeedback(t.TempDir())
	if len(fb.FailedTitles) != 0 {
		t.Errorf("expected empty feedback; got %+v", fb)
	}
}

func TestFeedback_StringEmbedsList(t *testing.T) {
	fb := Feedback{FailedTitles: []string{"a", "b"}}
	got := fb.String()
	if !strings.Contains(got, "DO NOT propose") {
		t.Errorf("missing avoid-this directive: %s", got)
	}
	if !strings.Contains(got, "- a") || !strings.Contains(got, "- b") {
		t.Errorf("missing failed titles: %s", got)
	}
}

func TestFeedback_EmptyStringWhenNoFailures(t *testing.T) {
	if got := (Feedback{}).String(); got != "" {
		t.Errorf("expected empty string; got %q", got)
	}
}
