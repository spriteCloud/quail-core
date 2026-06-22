package mindmap

import "testing"

func TestJourneyPriority(t *testing.T) {
	cases := map[JourneyKind]string{
		JourneyConvert:      "critical",
		JourneyContact:      "critical",
		JourneyAuthenticate: "critical",
		JourneyEvaluate:     "standard",
		JourneyResearch:     "standard",
		JourneyBrowse:       "standard",
		JourneyDiscover:     "standard",
		JourneyExercise:     "standard",
		JourneyExplore:      "nice-to-have",
		JourneyRead:         "nice-to-have",
	}
	for k, want := range cases {
		if got := JourneyPriority(k); got != want {
			t.Errorf("JourneyPriority(%q) = %q; want %q", k, got, want)
		}
	}
	if got := JourneyPriority("unknown-kind"); got != "standard" {
		t.Errorf("unknown kind should default to 'standard'; got %q", got)
	}
}
