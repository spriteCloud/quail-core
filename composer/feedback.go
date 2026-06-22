package composer

import (
	"os"
	"strings"
)

// Feedback is the subset of the bug-discovery ledger
// (tests/e2e/docs/findings.md) that the composer reads to AVOID
// re-proposing scenarios that have repeatedly failed.
type Feedback struct {
	// FailedTitles is the list of scenario / test titles that have
	// appeared as failures in the ledger. The composer's prompt is
	// extended with an explicit "do not propose any of these" list.
	FailedTitles []string
}

// LoadFeedback reads tests/e2e/docs/findings.md from the workdir and
// extracts failed test titles. Missing file / malformed table is not
// an error — empty Feedback returned in that case.
func LoadFeedback(workDir string) Feedback {
	for _, candidate := range []string{
		workDir + "/tests/e2e/docs/findings.md",
		"tests/e2e/docs/findings.md",
	} {
		body, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		return parseFeedback(string(body))
	}
	return Feedback{}
}

func parseFeedback(body string) Feedback {
	var fb Feedback
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") || !strings.HasSuffix(t, "|") {
			continue
		}
		// Skip header / separator rows.
		if strings.HasPrefix(t, "|---") || strings.Contains(t, "| Spec |") || strings.Contains(t, "|---|") {
			continue
		}
		// Split, find the "Test" column (index 1 in the canonical layout).
		parts := strings.Split(strings.Trim(t, "|"), "|")
		if len(parts) < 2 {
			continue
		}
		title := strings.TrimSpace(parts[1])
		if title == "" {
			continue
		}
		fb.FailedTitles = append(fb.FailedTitles, title)
	}
	return fb
}

// String renders the feedback as a prompt fragment the composer
// appends to its user message. Empty fragment when there's no
// feedback to incorporate.
func (f Feedback) String() string {
	if len(f.FailedTitles) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nScenarios that have repeatedly failed against this app — DO NOT propose anything substantially similar:\n")
	for _, t := range f.FailedTitles {
		b.WriteString("  - ")
		b.WriteString(t)
		b.WriteString("\n")
	}
	return b.String()
}
