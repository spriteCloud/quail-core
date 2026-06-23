package composer

import (
	"regexp"
	"strings"
	"testing"
)

// TestBuildSystemPrompt_NilFallback locks in back-compat: when no
// patterns are supplied the system prompt matches the legacy hardcoded
// const byte-for-byte. Callers that haven't migrated must see no
// observable change.
func TestBuildSystemPrompt_NilFallback(t *testing.T) {
	got := buildSystemPrompt(nil)
	if got != systemPrompt {
		t.Fatalf("buildSystemPrompt(nil) != legacy systemPrompt\n--- got len=%d ---\n%s\n--- want len=%d ---\n%s",
			len(got), got, len(systemPrompt), systemPrompt)
	}
}

// TestBuildSystemPrompt_DynamicIncludesPatterns checks the dynamic
// branch puts each supplied pattern into the prompt's patterns block.
func TestBuildSystemPrompt_DynamicIncludesPatterns(t *testing.T) {
	patterns := []StepPattern{
		{Keyword: "Then", Raw: `/^the URL contains "([^"]+)"$/`, Match: regexp.MustCompile(`the URL contains "([^"]+)"`)},
		{Keyword: "Given", Raw: `'I open the landing page'`, Match: regexp.MustCompile(`^I open the landing page$`)},
		{Keyword: "Then", Raw: `/^there are (\d+) "([^"]+)" elements$/`, Match: regexp.MustCompile(`there are (\d+) "([^"]+)" elements`)},
	}
	got := buildSystemPrompt(patterns)
	for _, want := range []string{
		`Then the URL contains "<param>"`,
		`Given I open the landing page`,
		`Then there are <n> "<param>" elements`,
		`assertion variety`, // the new nudge
	} {
		if !strings.Contains(got, want) {
			t.Errorf("buildSystemPrompt output missing %q", want)
		}
	}
}
