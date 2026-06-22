package probe

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSuiteAlreadyCovers(t *testing.T) {
	tmp := t.TempDir()
	urls := []string{"https://example.com"}

	// No marker → false.
	if SuiteAlreadyCovers(tmp, urls, time.Now()) {
		t.Fatalf("missing marker should not cover")
	}

	// Fresh marker → true.
	if err := WriteState(tmp, urls, "vTEST"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !SuiteAlreadyCovers(tmp, urls, time.Now()) {
		t.Errorf("fresh marker should cover")
	}

	// Marker older than TTL → false.
	if !SuiteAlreadyCovers(tmp, urls, time.Now().Add(time.Hour)) {
		t.Errorf("1h-old marker should still cover")
	}
	if SuiteAlreadyCovers(tmp, urls, time.Now().Add(stateTTL+time.Hour)) {
		t.Errorf("TTL+1h marker should NOT cover")
	}

	// Different URLs → false.
	if SuiteAlreadyCovers(tmp, []string{"https://other.com"}, time.Now()) {
		t.Errorf("different URLs should not cover")
	}

	// Order-independence + duplicates collapsed.
	if err := WriteState(tmp, []string{"https://a.com", "https://b.com"}, "vTEST"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !SuiteAlreadyCovers(tmp, []string{"https://b.com", "https://a.com"}, time.Now()) {
		t.Errorf("order should not matter")
	}

	// Corrupt marker → treat as missing.
	stateFile := filepath.Join(tmp, stateRelPath)
	if err := os.WriteFile(stateFile, []byte("garbage{{}}"), 0o644); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if SuiteAlreadyCovers(tmp, urls, time.Now()) {
		t.Errorf("corrupt marker should not cover")
	}
}
