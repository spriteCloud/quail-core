package gen

import (
	"testing"

	"github.com/spriteCloud/quail-core/ast"
)

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

// v0.95.6: inferInputType uses field name + label + aria + placeholder
// hints to pick the *effective* type when the input wasn't tagged
// with an HTML5 type. Drives paramRowsFor (which Examples to emit)
// and realisticValueFor (the happy-compute Scenario's input values).
func TestInferInputType(t *testing.T) {
	cases := []struct {
		name string
		in   ast.FormInput
		want string
	}{
		// Specific type passes through.
		{"explicit number", ast.FormInput{Type: "number"}, "number"},
		{"explicit email", ast.FormInput{Type: "email"}, "email"},
		// Empty type, name hints number.
		{"monthlyIncome", ast.FormInput{Name: "monthlyIncome"}, "number"},
		{"bruto_jaarinkomen NL", ast.FormInput{Name: "bruto_jaarinkomen"}, "number"},
		{"label says income", ast.FormInput{LabelText: "Jouw bruto jaarinkomen"}, "number"},
		{"loan amount", ast.FormInput{Name: "loanAmount"}, "number"},
		{"hypotheek amount NL", ast.FormInput{Aria: "Hypotheek"}, "number"},
		{"montant FR", ast.FormInput{Placeholder: "Montant"}, "number"},
		// Empty type, name hints email.
		{"email_address", ast.FormInput{Name: "email_address"}, "email"},
		{"correo ES", ast.FormInput{Name: "correo"}, "email"},
		// Empty type, name hints tel.
		{"customer_phone", ast.FormInput{Name: "customer_phone"}, "tel"},
		{"telefoon NL", ast.FormInput{Name: "telefoon"}, "tel"},
		// Empty type, name hints date.
		{"dob", ast.FormInput{Name: "dob"}, "date"},
		{"checkin", ast.FormInput{Name: "checkin"}, "date"},
		{"datum NL", ast.FormInput{LabelText: "Geboortedatum"}, "date"},
		// Empty type, name hints postcode.
		{"postal_code", ast.FormInput{Name: "postal_code"}, "postcode"},
		{"zip", ast.FormInput{Name: "zip"}, "postcode"},
		{"cep BR", ast.FormInput{Name: "cep"}, "postcode"},
		// Empty type, no hint → text.
		{"username unknown", ast.FormInput{Name: "username"}, "text"},
		{"completely empty", ast.FormInput{}, "text"},
		// Select pass-through.
		{"select", ast.FormInput{Type: "select"}, "select"},
	}
	for _, tc := range cases {
		got := inferInputType(tc.in)
		if got != tc.want {
			t.Errorf("%s: inferInputType = %q, want %q", tc.name, got, tc.want)
		}
	}
}
