package llm

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare", "http://host:11434", "http://host:11434/v1"},
		{"bare-trailing-slash", "http://host:11434/", "http://host:11434/v1"},
		{"already-v1", "http://host:11434/v1", "http://host:11434/v1"},
		{"already-v1-trailing-slash", "http://host:11434/v1/", "http://host:11434/v1"},
		{"surrounding-whitespace", "  http://host:11434/v1  ", "http://host:11434/v1"},
		{"empty", "", ""},
		{"whitespace-only", "   ", ""},
		{"openai", "https://api.openai.com/v1", "https://api.openai.com/v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeBaseURL(tc.in)
			if got != tc.want {
				t.Fatalf("NormalizeBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
