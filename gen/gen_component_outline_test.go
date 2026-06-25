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
		// humanField cascades Label/Aria/Placeholder/humanize(Name) →
		// the fixture has only Name set, so the display is the
		// humanised form ("bruto-jaarinkomen" → "Bruto jaarinkomen").
		display := humanizeIdentifier(f.Name)
		title := `Scenario Outline: fill the "` + display + `" field with <example>`
		if !strings.Contains(body, title) {
			t.Errorf("expected outline title %q in feature output", title)
		}
		step := `When I enter "<example>" into the "` + display + `" field`
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

	// v0.95.6: ONE happy-compute Scenario per PrimaryComponent. Fills
	// every input with a realistic value and asserts a result appears.
	mustContain(t, body, "@priority:critical @kind:component-submit")
	mustContain(t, body, "Scenario: filling the primary component yields a result")
	mustContain(t, body, "Then a result is shown")

	// v0.10.12: sad-path siblings to the happy-compute Scenario.
	// Each closes one obvious gap that v0.95.6 left open.
	mustContain(t, body, "@kind:component-empty")
	mustContain(t, body, "Scenario: empty submit on the primary component is rejected")
	mustContain(t, body, "Then a validation hint is shown")

	mustContain(t, body, "@kind:component-invalid")
	mustContain(t, body, `When I enter "not-a-number" into the "Bruto jaarinkomen" field`)
	mustContain(t, body, "Then no result is shown")

	mustContain(t, body, "@kind:component-partial")
	mustContain(t, body, "Then the page does not crash")

	// All-numeric component (two number inputs + a select). Select
	// breaks the all-numeric gate → boundary Outline must NOT fire.
	if strings.Contains(body, "@kind:component-boundary") {
		t.Errorf("boundary Outline fired despite a non-numeric input (select):\n%s", body)
	}
}

// v0.10.12: when every primary-component input is numeric (the ING
// hypotheek case with no select dropdown), the boundary Outline DOES
// fire — three rows: zero, max-realistic, negative.
func TestFeatureTemplate_PrimaryComponent_BoundaryOutlineForAllNumeric(t *testing.T) {
	pc := &ast.PrimaryComponent{
		Selector: "flex-calc",
		Inputs: []ast.FormInput{
			{Name: "applicant", LabelText: "Jouw bruto jaarinkomen"},
			{Name: "partner", LabelText: "Bruto jaarinkomen partner"},
		},
	}
	landing := ast.Symbol{
		Name: "X", Kind: ast.KindComponent,
		File: "https://x.test/", Language: "ts",
		HasForm: true, Inputs: pc.Inputs, PrimaryComponent: pc,
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
	mustContain(t, body, "@kind:component-boundary")
	mustContain(t, body, "Scenario Outline: primary component handles boundary numeric values")
	mustContain(t, body, "| zero          | 0      |")
	mustContain(t, body, "| max-realistic | 500000 |")
	mustContain(t, body, "| negative      | -1     |")
}

// v0.95.6: when the primary component holds inputs the probe
// captured WITHOUT an HTML5 type (the ING flex-component case), the
// name-hint inference turns them into number rows — not the
// text-junk row pool. The 3 Outlines must use number-case rows.
func TestFeatureTemplate_PrimaryComponent_InferredNumberRowsForIncomeFields(t *testing.T) {
	pc := &ast.PrimaryComponent{
		Selector: "flex-calc",
		Inputs: []ast.FormInput{
			// All three lack an explicit Type but have income-shaped labels.
			{Name: "applicant", LabelText: "Jouw bruto jaarinkomen"},
			{Name: "partner", LabelText: "Bruto jaarinkomen partner"},
			{Name: "monthlyIncome", LabelText: "Bruto maandbedrag"},
		},
	}
	landing := ast.Symbol{
		Name: "ING", Kind: ast.KindComponent,
		File: "https://x.test/calc", Language: "ts",
		PageTitle: "Calc", HasForm: true,
		Inputs: pc.Inputs, PrimaryComponent: pc,
	}
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/ing.feature", JourneyKind: "exercise",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	// Number-row markers — NOT the text-junk row markers.
	for _, want := range []string{"| zero | 0 |", "| typical | 42 |", "| joint-income | 60000 |"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected number-case row %q in feature output\nbody:\n%s", want, body)
		}
	}
	for _, junk := range []string{"café-niño-ümlaut", "hello 🎉 world 🌍", "rtl-mark"} {
		if strings.Contains(body, junk) {
			t.Errorf("text-junk row %q must NOT appear for number-inferred fields", junk)
		}
	}
	// Happy-compute Scenario uses realistic value (50000).
	mustContain(t, body, `When I enter "50000" into the "Jouw bruto jaarinkomen" field`)
	mustContain(t, body, `When I enter "50000" into the "Bruto maandbedrag" field`)
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
