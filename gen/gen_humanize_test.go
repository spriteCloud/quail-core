package gen

import "testing"

func TestHumanizeIdentifier(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Empty stays empty.
		{"", ""},

		// camelCase → Title case.
		{"monthlyIncome", "Monthly income"},
		{"firstName", "First name"},
		{"bruto", "Bruto"},

		// snake_case → Title case.
		{"bruto_jaarinkomen", "Bruto jaarinkomen"},
		{"phone_number_e164", "Phone number e164"},

		// kebab-case → Title case.
		{"bruto-jaarinkomen", "Bruto jaarinkomen"},
		{"opt-in-marketing", "Opt in marketing"},

		// SCREAMING_SNAKE → Title case.
		{"MAX_RETRIES", "Max retries"},

		// PascalCase → Title case.
		{"MonthlyIncome", "Monthly income"},
		{"ZipCode", "Zip code"},

		// Already human (contains space) → leave alone.
		{"Monthly income", "Monthly income"},
		{"  has  spaces  ", "  has  spaces  "}, // spaces present → no rewrite

		// Single letter / acronym boundaries.
		{"x", "X"},
		// ponytail: PascalCase with an all-caps prefix becomes "Url path"
		// not "URL path" — we don't try to preserve acronyms because
		// distinguishing them from SCREAMING_SNAKE words isn't worth
		// the heuristic complexity for our calculator-field use case.
		{"URLPath", "Url path"},
	}
	for _, tc := range cases {
		got := humanizeIdentifier(tc.in)
		if got != tc.want {
			t.Errorf("humanizeIdentifier(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
