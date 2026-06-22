package probe

import (
	"context"
	"testing"
)

func TestParseCoverage(t *testing.T) {
	cases := map[string]CoverageMode{
		"":                  CoverageStandard,
		"standard":          CoverageStandard,
		"breadth":           CoverageBreadth,
		"depth":             CoverageDepth,
		"DEPTH":             CoverageDepth,
		"  breadth  ":       CoverageBreadth,
		"unknown-mode":      CoverageStandard, // defaults to standard
	}
	for raw, want := range cases {
		if got := ParseCoverage(raw); got != want {
			t.Errorf("ParseCoverage(%q) = %q; want %q", raw, got, want)
		}
	}
}

func TestCoverageMode_DialsTheKnobs(t *testing.T) {
	// Knobs must scale: breadth < standard < depth across all three dials.
	for _, k := range []struct {
		name string
		fn   func(CoverageMode) int
	}{
		{"MaxPages", func(c CoverageMode) int { return c.crawlOpts().MaxPages }},
		{"MaxDepth", func(c CoverageMode) int { return c.crawlOpts().MaxDepth }},
		{"JourneysPerKind", func(c CoverageMode) int { return c.JourneysPerKind() }},
		{"FuzzCap", func(c CoverageMode) int { return c.FuzzCap() }},
	} {
		b, s, d := k.fn(CoverageBreadth), k.fn(CoverageStandard), k.fn(CoverageDepth)
		if !(b < s && s < d) {
			t.Errorf("%s should scale breadth(%d) < standard(%d) < depth(%d)", k.name, b, s, d)
		}
	}
}

// v0.90: CoverageMax used to fall through to the default 3 in
// JourneysPerKind (silent typo — `--coverage max` raised MaxPages
// but not journey count). Pin the rise so a future merge can't
// re-introduce the regression.
func TestCoverageMax_RaisesJourneyAndFuzzCaps(t *testing.T) {
	if got := CoverageMax.JourneysPerKind(); got <= CoverageDepth.JourneysPerKind() {
		t.Errorf("CoverageMax JourneysPerKind = %d; must exceed depth's %d", got, CoverageDepth.JourneysPerKind())
	}
	if got := CoverageMax.FuzzCap(); got <= CoverageDepth.FuzzCap() {
		t.Errorf("CoverageMax FuzzCap = %d; must exceed depth's %d", got, CoverageDepth.FuzzCap())
	}
}

func TestParseMaxJourneys(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"abc", 0},
		{"-1", 0},
		{"0", 0},
		{"5", 5},
		{"50", 50},
	}
	for _, tc := range cases {
		if got := ParseMaxJourneys(tc.raw); got != tc.want {
			t.Errorf("ParseMaxJourneys(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestParseMaxJourneys_HonoursEnv(t *testing.T) {
	t.Setenv("QUAIL_MAX_JOURNEYS", "7")
	if got := ParseMaxJourneys(""); got != 7 {
		t.Errorf("env override: ParseMaxJourneys(\"\") = %d, want 7", got)
	}
	if got := ParseMaxJourneys("2"); got != 2 {
		t.Errorf("explicit flag should beat env: ParseMaxJourneys(\"2\") = %d, want 2", got)
	}
}

func TestMaxJourneys_CtxRoundTrip(t *testing.T) {
	ctx := WithMaxJourneys(context.Background(), 42)
	if got := maxJourneysFromCtx(ctx); got != 42 {
		t.Errorf("ctx round-trip: %d, want 42", got)
	}
	if got := maxJourneysFromCtx(context.Background()); got != 0 {
		t.Errorf("no override on bare ctx: %d, want 0", got)
	}
}
