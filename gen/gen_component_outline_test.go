package gen

import (
	"strings"
	"testing"

	"github.com/spriteCloud/quail-core/ast"
	"github.com/spriteCloud/quail-core/plan"
)

// ING-shape calculator: the page carries a PrimaryComponent with 3
// inputs (two number, one select-with-options). The feature template
// must emit one Scenario Outline per input, each with its own
// Examples table. This is what makes a mortgage calculator more
// useful than "Probe a URL → 1 scenario".
func TestFeatureTemplate_PrimaryComponent_FansOutPerInput(t *testing.T) {
	pc := &ast.PrimaryComponent{
		Selector: "flex-calc",
		Inputs: []ast.FormInput{
			{Name: "bruto-jaarinkomen", Type: "number"},
			{Name: "partner-inkomen", Type: "number"},
			{Name: "energielabel", Type: "select", OptionValues: []string{"A", "B", "C", "D", "E", "F", "G"}},
		},
	}
	landing := ast.Symbol{
		Name: "ING", Kind: ast.KindComponent,
		File: "https://www.ing.nl/particulier/hypotheek/hypotheek-berekenen",
		Language:  "ts",
		PageTitle: "Hypotheek berekenen",
		HasForm:   true,
		Inputs:    pc.Inputs,
		PrimaryComponent: pc,
	}
	it := plan.Item{
		Symbol:      landing,
		Symbols:     []ast.Symbol{landing},
		PageURL:     landing.File,
		Template:    plan.TmplPlaywrightFeature,
		OutPath:     "tests/e2e/features/ing-berekenen.feature",
		JourneyKind: "convert",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)

	// One Scenario Outline per input on the primary component. The
	// title uses friendly English so the raw Gherkin reads cleanly
	// in the platform UI even before per-row substitution — no
	// `<variant>` / `<value>` placeholders escape into the title.
	for _, f := range pc.Inputs {
		want := "@field:" + f.Name
		if !strings.Contains(body, want) {
			t.Errorf("expected %q tag in feature output", want)
		}
		title := `Scenario Outline: fill the "` + f.Name + `" field with <example>`
		if !strings.Contains(body, title) {
			t.Errorf("expected outline title %q in feature output", title)
		}
		step := `When I enter "<example>" into the "` + f.Name + `" field`
		if !strings.Contains(body, step) {
			t.Errorf("expected step %q in feature output", step)
		}
	}

	// Examples table uses the new friendlier column headers.
	if !strings.Contains(body, "| case | example |") {
		t.Errorf("expected friendly Examples header `| case | example |`\nbody:\n%s", body)
	}

	// Selector (`flex-calc`) must NOT leak into the user-facing title.
	if strings.Contains(body, "flex-calc —") || strings.Contains(body, "Scenario Outline: flex-calc") {
		t.Errorf("primary-component selector must stay out of the title\nbody:\n%s", body)
	}

	// Select rows: each captured <option> becomes one Examples row.
	for _, v := range pc.Inputs[2].OptionValues {
		row := "| " + strings.ToLower(v) + " | " + v + " |"
		if !strings.Contains(body, row) {
			t.Errorf("expected select Examples row %q in feature output\nbody:\n%s", row, body)
		}
	}

	// All three Outlines must be tagged @kind:component so the kinds
	// picker can surface or filter them.
	count := strings.Count(body, "@kind:component")
	if count < 3 {
		t.Errorf("expected ≥3 @kind:component scenarios for 3 inputs, got %d\nbody:\n%s", count, body)
	}
}

// Two inputs sharing a (Name, Type) — common when a calculator
// page renders a hidden mirror alongside the visible field — must
// produce exactly one Outline downstream. The probe-Go layer
// dedupes before the template sees the duplicate.
func TestFeatureTemplate_PrimaryComponent_DedupesByName(t *testing.T) {
	pc := &ast.PrimaryComponent{
		Selector: "flex-calc",
		Inputs: []ast.FormInput{
			{Name: "monthlyIncome", Type: "number"},
			{Name: "monthlyIncome", Type: "number"}, // duplicate; should be dropped by probe.
		},
	}
	// Manually dedupe here too — the test exercises the template
	// behaviour given a deduped slice, since the probe-Go dedupe
	// is covered by browser_probe_test.go. Templates must NOT
	// reintroduce duplicates regardless.
	landing := ast.Symbol{
		Name: "X", Kind: ast.KindComponent,
		File: "https://x.test/", Language: "ts",
		PageTitle:        "Calculator",
		HasForm:          true,
		Inputs:           pc.Inputs[:1],
		PrimaryComponent: &ast.PrimaryComponent{Selector: pc.Selector, Inputs: pc.Inputs[:1]},
	}
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/x.feature", JourneyKind: "exercise",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	count := strings.Count(body, "@field:monthlyIncome")
	if count != 1 {
		t.Errorf("expected exactly 1 outline for monthlyIncome, got %d\nbody:\n%s", count, body)
	}
}

// A page without a primary component (no calculator, no <main> form,
// no shadow-DOM widget) must NOT emit any component-scoped outlines.
// The existing firstTextInput path stays intact.
func TestFeatureTemplate_NoPrimaryComponent_SkipsFanOut(t *testing.T) {
	landing := ast.Symbol{
		Name: "Marketing", Kind: ast.KindComponent,
		File: "https://example.com/", Language: "ts",
		PageTitle: "Home",
		HasForm:   true,
		Inputs: []ast.FormInput{
			{Name: "email", Type: "email", Required: true},
		},
		// PrimaryComponent left nil.
	}
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/marketing.feature", JourneyKind: "convert",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	if strings.Contains(body, "@kind:component") {
		t.Errorf("nil PrimaryComponent must not emit component outlines:\n%s", body)
	}
}
