package gen

import (
	"strings"
	"testing"

	"github.com/spriteCloud/quail-core/ast"
	"github.com/spriteCloud/quail-core/plan"
)

// v0.10.13: when the LLM classified this page as a "calculator" with
// confidence >= 0.7, the feature template emits BOTH the generic
// component scenarios AND the calculator-relational scenario.
func TestFeatureTemplate_CalculatorIntent_EmitsRelationalScenario(t *testing.T) {
	body := renderWithIntent(t, ast.PageIntent{Intent: "calculator", Confidence: 0.9})
	mustContain(t, body, "@kind:calculator-relational")
	mustContain(t, body, "Scenario: the result reacts when an input value changes")
	mustContain(t, body, "When I capture the current result")
	mustContain(t, body, "Then the result has changed")
	// The realisticAltValueFor (100000) must appear — proves we
	// refilled with a different value, not the same realisticValueFor.
	mustContain(t, body, `When I enter "100000" into the "Applicant" field`)
	// The generic scenarios must still emit — the calculator template
	// includes them via {{ template "component_generic" . }}.
	mustContain(t, body, "@kind:component-submit")
	mustContain(t, body, "@kind:component-empty")
}

// Below the 0.7 confidence floor → fall back to generic. The
// calculator-relational scenario must NOT appear.
func TestFeatureTemplate_CalculatorIntent_LowConfidence_FallsBackToGeneric(t *testing.T) {
	body := renderWithIntent(t, ast.PageIntent{Intent: "calculator", Confidence: 0.5})
	if strings.Contains(body, "@kind:calculator-relational") {
		t.Errorf("low-confidence intent must not trigger specialization:\n%s", body)
	}
	mustContain(t, body, "@kind:component-submit")
}

// Empty intent (LLM-off, or classifier returned unknown) → generic
// fires; output is byte-identical to v0.10.12.
func TestFeatureTemplate_NoIntent_EmitsGenericOnly(t *testing.T) {
	body := renderWithIntent(t, ast.PageIntent{})
	if strings.Contains(body, "@kind:calculator-relational") {
		t.Errorf("zero intent must not trigger specialization:\n%s", body)
	}
	mustContain(t, body, "@kind:component-submit")
	mustContain(t, body, "@kind:component-empty")
	mustContain(t, body, "@kind:component-invalid")
}

// Unrelated intent (e.g. booking) at high confidence → generic
// scenarios fire; no calculator-relational scenario. Other intent
// templates aren't wired in PR1.
func TestFeatureTemplate_NonCalculatorIntent_FallsBackToGeneric(t *testing.T) {
	body := renderWithIntent(t, ast.PageIntent{Intent: "booking", Confidence: 0.95})
	if strings.Contains(body, "@kind:calculator-relational") {
		t.Errorf("non-calculator intent must not trigger calculator template:\n%s", body)
	}
	mustContain(t, body, "@kind:component-submit")
}

func renderWithIntent(t *testing.T, intent ast.PageIntent) string {
	t.Helper()
	pc := &ast.PrimaryComponent{
		Selector: "flex-calc",
		Inputs: []ast.FormInput{
			{Name: "applicant", LabelText: "Applicant", Type: "number"},
			{Name: "partner", LabelText: "Partner", Type: "number"},
		},
	}
	landing := ast.Symbol{
		Name: "ING", Kind: ast.KindComponent,
		File: "https://www.ing.nl/hypotheek/berekenen", Language: "ts",
		PageTitle:        "Hypotheek berekenen",
		HasForm:          true,
		Inputs:           pc.Inputs,
		PrimaryComponent: pc,
		PageIntent:       intent,
	}
	it := plan.Item{
		Symbol:      landing,
		Symbols:     []ast.Symbol{landing},
		PageURL:     landing.File,
		Template:    plan.TmplPlaywrightFeature,
		OutPath:     "tests/e2e/features/x.feature",
		JourneyKind: "exercise",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return string(out[0].Content)
}
