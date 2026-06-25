package gen

import (
	"strings"
	"testing"

	"github.com/spriteCloud/quail-core/ast"
	"github.com/spriteCloud/quail-core/plan"
)

func TestFeatureTemplate_EmitsTaggedScenario(t *testing.T) {
	landing := ast.Symbol{
		Name: "X", Kind: ast.KindComponent, File: "https://x.test/", Language: "ts",
		PageTitle: "Home",
		HasForm:   true,
		Inputs: []ast.FormInput{
			{Name: "email", Type: "email", Required: true},
		},
	}
	it := plan.Item{
		Symbol:      landing,
		Symbols:     []ast.Symbol{landing},
		PageURL:     "https://x.test/",
		Template:    plan.TmplPlaywrightFeature,
		OutPath:     "tests/e2e/features/x-convert.feature",
		JourneyKind: "convert",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, "Feature: X — convert · /")
	mustContain(t, body, "@journey:convert @priority:critical @smoke")
	mustContain(t, body, "Scenario: convert journey reaches its terminal page")
	mustContain(t, body, "Given I open the landing page")
	mustContain(t, body, `And I enter "test@example.com" into the "Email" field`)
	mustContain(t, body, "And I submit the form")
	mustContain(t, body, "@journey:convert @priority:critical @negative")
	mustContain(t, body, "Scenario: convert — empty submission is rejected")
	// Sanity: a feature file must not contain TypeScript runtime symbols.
	if strings.Contains(body, "import {") || strings.Contains(body, "test.describe") {
		t.Errorf("feature file should be pure Gherkin, not TS:\n%s", body)
	}
}

func TestFeatureTemplate_ChainedJourneyEmitsWhenSteps(t *testing.T) {
	landing := ast.Symbol{Name: "X", Kind: ast.KindComponent, File: "https://x.test/", Language: "ts", PageTitle: "Home"}
	step2 := ast.Symbol{
		Name: "X", Kind: ast.KindComponent, File: "https://x.test/blog", Language: "ts",
		PageTitle: "Blog", EnteredVia: "/blog",
	}
	it := plan.Item{
		Symbol:      landing,
		Symbols:     []ast.Symbol{landing, step2},
		PageURL:     "https://x.test/",
		Template:    plan.TmplPlaywrightFeature,
		OutPath:     "tests/e2e/features/x-browse.feature",
		JourneyKind: "browse",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, "@journey:browse @priority:standard @smoke")
	mustContain(t, body, `When I click the link to "/blog"`)
	mustContain(t, body, `And the page title contains "Blog"`)
}

// v0.95: the Feature: title carries the URL path so the platform
// sidebar can distinguish "browse · /particulier" from
// "browse · /zakelijk" at a glance instead of rendering ten rows of
// identical "X — browse journey" labels.
func TestFeatureTemplate_TitleIncludesURLPath(t *testing.T) {
	landing := ast.Symbol{
		Name: "test3", Kind: ast.KindComponent,
		File: "https://www.example.com/foo/bar", Language: "ts",
		PageTitle: "Foo bar page",
	}
	it := plan.Item{
		Symbol:      landing,
		Symbols:     []ast.Symbol{landing},
		PageURL:     landing.File,
		Template:    plan.TmplPlaywrightFeature,
		OutPath:     "tests/e2e/features/x.feature",
		JourneyKind: "browse",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	want := "Feature: test3 — browse · /foo/bar"
	if !strings.Contains(body, want) {
		t.Errorf("expected title %q, got body:\n%s", want, body)
	}
	// And the redundant " journey" suffix must be gone.
	if strings.Contains(body, "browse journey\n") {
		t.Errorf("Feature title should drop the redundant ' journey' suffix, got:\n%s", body)
	}
}

// v0.95.1: when a probed input carries a visible <label> text, the
// Outline / step text reads the label — not the `name` slug. Tag
// stays on Name for grep-ability.
func TestFeatureTemplate_HumanField_PrefersLabelText(t *testing.T) {
	landing := ast.Symbol{
		Name: "X", Kind: ast.KindComponent,
		File: "https://x.test/calc", Language: "ts",
		PageTitle: "Calc",
		HasForm:   true,
		Inputs: []ast.FormInput{
			{Name: "monthlyIncome", Type: "number", LabelText: "Monthly income"},
		},
		PrimaryComponent: &ast.PrimaryComponent{
			Selector: "calc",
			Inputs: []ast.FormInput{
				{Name: "monthlyIncome", Type: "number", LabelText: "Monthly income"},
			},
		},
	}
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/calc.feature", JourneyKind: "exercise",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, `Scenario Outline: fill the "Monthly income" field with <example>`)
	mustContain(t, body, `When I enter "<example>" into the "Monthly income" field`)
	mustContain(t, body, `@field:monthlyIncome`)
	if strings.Contains(body, `"monthlyIncome" field`) {
		t.Errorf("expected step text to use LabelText, not Name slug:\n%s", body)
	}
}

// When LabelText / Aria / Placeholder are all empty, fall back to a
// humanised version of Name ("monthlyIncome" → "Monthly income"). The
// step-def fillForm cascades through label/aria/name candidates so
// the humanised title still resolves to <input name="monthlyIncome">.
// v0.95.2.
func TestFeatureTemplate_HumanField_FallsBackToHumanizedName(t *testing.T) {
	landing := ast.Symbol{
		Name: "X", Kind: ast.KindComponent,
		File: "https://x.test/calc", Language: "ts",
		PageTitle: "Calc",
		HasForm:   true,
		Inputs: []ast.FormInput{
			{Name: "monthlyIncome", Type: "number"},
		},
		PrimaryComponent: &ast.PrimaryComponent{
			Selector: "calc",
			Inputs: []ast.FormInput{
				{Name: "monthlyIncome", Type: "number"},
			},
		},
	}
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/calc.feature", JourneyKind: "exercise",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, `Scenario Outline: fill the "Monthly income" field with <example>`)
	mustContain(t, body, `@field:monthlyIncome`)
}

func TestStepDefsBDDTemplate_BindsToStepsAPI(t *testing.T) {
	sym := ast.Symbol{Name: "X", Kind: ast.KindComponent, File: "https://x.test", Language: "ts"}
	it := plan.Item{
		Symbol:   sym,
		Symbols:  []ast.Symbol{sym},
		PageURL:  "https://x.test",
		Template: plan.TmplPlaywrightStepsBDD,
		OutPath:  "tests/e2e/steps/quail.steps.ts",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, `import { createBdd } from 'playwright-bdd'`)
	mustContain(t, body, `import { steps } from '../lib/steps'`)
	mustContain(t, body, "const { Given, When, Then } = createBdd()")
	// Coverage of the vocabulary the feature template emits.
	for _, pattern := range []string{
		`Given('I open the landing page'`,
		`When(/^I click the link to "([^"]+)"$/`,
		`When(/^I navigate directly to "([^"]+)"$/`,
		`When(/^I enter "([^"]+)" into the "([^"]+)" field$/`,
		`When('I submit the form'`,
		`When('I submit the form without filling any required field'`,
		`Then(/^the page title contains "([^"]+)"$/`,
		`Then(/^the main heading reads "([^"]+)"$/`,
		`Then(/^I see the heading "([^"]+)"$/`,
		`Then('no error message is shown in the form region'`,
		`Then('I remain on the same page'`,
		`Then('no success message is shown'`,
		`Then(/^the URL contains "([^"]+)"$/`,
		`Then(/^the page has at least (\d+) items$/`,
		`When('I scroll to the bottom of the page'`,
		`When(/^I press the "([^"]+)" key$/`,
	} {
		if !strings.Contains(body, pattern) {
			t.Errorf("steps file missing pattern %q", pattern)
		}
	}
}

func TestConfigTemplate_UsesDefineBddConfig(t *testing.T) {
	sym := ast.Symbol{Name: "X", Kind: ast.KindComponent, File: "https://x.test", Language: "ts"}
	it := plan.Item{
		Symbol:   sym,
		Symbols:  []ast.Symbol{sym},
		PageURL:  "https://x.test",
		Template: plan.TmplPlaywrightConfig,
		OutPath:  "playwright.config.ts",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, "import { defineBddConfig } from 'playwright-bdd'")
	mustContain(t, body, "defineBddConfig({")
	mustContain(t, body, "features: 'tests/e2e/features/*.feature'")
	mustContain(t, body, "steps: 'tests/e2e/steps/*.ts'")
	// v0.22: project naming switched to "<project>-<browser>" + the
	// testMatch grew to cover the quality companions.
	mustContain(t, body, "`bdd-${browser}`")
	mustContain(t, body, "`extras-${browser}`")
	mustContain(t, body, "api|fuzz|a11y|responsive|perf|security|health|observability|contract|i18n")
}

func TestPackageTemplate_DepsListPlaywrightBdd(t *testing.T) {
	sym := ast.Symbol{Name: "X", Kind: ast.KindComponent, File: "https://x.test", Language: "ts"}
	it := plan.Item{
		Symbol:   sym,
		Symbols:  []ast.Symbol{sym},
		PageURL:  "https://x.test",
		Template: plan.TmplPlaywrightPackage,
		OutPath:  "package.json",
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := string(out[0].Content)
	mustContain(t, body, `"playwright-bdd"`)
	mustContain(t, body, `"test:critical"`)
}
