// Package heal repairs broken Playwright locators. The default trigger is a
// failing Playwright JSON report; proactive mode scans diffed UI files for
// anchors that moved.
package heal

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// PlaywrightReport is a permissive subset of the Playwright JSON reporter
// schema. Fields we do not consume are left out; unknown fields are ignored.
type PlaywrightReport struct {
	Suites []Suite `json:"suites"`
}

type Suite struct {
	Title  string  `json:"title"`
	File   string  `json:"file"`
	Suites []Suite `json:"suites"`
	Specs  []Spec  `json:"specs"`
}

type Spec struct {
	Title string `json:"title"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Tests []Test `json:"tests"`
}

type Test struct {
	Results []Result `json:"results"`
}

type Result struct {
	Status string `json:"status"`
	Errors []struct {
		Message string `json:"message"`
		Stack   string `json:"stack"`
		Snippet string `json:"snippet"`
	} `json:"errors"`
}

// Failure summarizes one failed test that probably hit a stale locator.
type Failure struct {
	File    string
	Line    int
	Title   string
	Locator string // best-effort extracted from error
	Reason  string // "timeout" | "strict-mode-violation" | "not-found" | "other"
}

func LoadReport(path string) (*PlaywrightReport, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r PlaywrightReport
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// LocatorFailures walks the report and emits one Failure per failed test
// whose error message contains a Playwright locator call.
func LocatorFailures(r *PlaywrightReport) []Failure {
	var out []Failure
	var walk func(s Suite)
	walk = func(s Suite) {
		for _, sp := range s.Specs {
			for _, t := range sp.Tests {
				for _, res := range t.Results {
					if res.Status == "passed" || res.Status == "skipped" {
						continue
					}
					for _, e := range res.Errors {
						msg := e.Message + "\n" + e.Stack + "\n" + e.Snippet
						loc := extractLocator(msg)
						if loc == "" {
							continue
						}
						out = append(out, Failure{
							File: chooseString(sp.File, s.File), Line: sp.Line,
							Title: sp.Title, Locator: loc, Reason: classify(msg),
						})
					}
				}
			}
		}
		for _, c := range s.Suites {
			walk(c)
		}
	}
	for _, s := range r.Suites {
		walk(s)
	}
	return out
}

func chooseString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

var (
	reGetBy = regexp.MustCompile(`(?:page|frame|locator|this|context)\.(getByRole|getByText|getByLabel|getByPlaceholder|getByAltText|getByTitle|getByTestId|locator)\(([^)]*)\)`)
)

func extractLocator(msg string) string {
	m := reGetBy.FindString(msg)
	return m
}

func classify(msg string) string {
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "timeout") || strings.Contains(low, "exceeded"):
		return "timeout"
	case strings.Contains(low, "strict mode violation"):
		return "strict-mode-violation"
	case strings.Contains(low, "no element") || strings.Contains(low, "not found") || strings.Contains(low, "not visible"):
		return "not-found"
	default:
		return "other"
	}
}
