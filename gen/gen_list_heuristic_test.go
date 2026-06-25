package gen

import (
	"strings"
	"testing"

	"github.com/spriteCloud/quail-core/ast"
	"github.com/spriteCloud/quail-core/plan"
)

// v0.10.11: empty-state Scenario is gated on pageIsListLike — it must
// NOT fire on section/marketing pages that happen to have a few h2s
// + links (ING /zakelijk is the canonical false positive).
func TestFeatureTemplate_NoEmptyState_OnSectionPage(t *testing.T) {
	contents := []ast.ContentAnchor{
		{Tag: "h1", Text: "Zakelijk"},
		{Tag: "h2", Text: "Diensten"},
		{Tag: "h2", Text: "Producten"},
	}
	links := []ast.LocatorAnchor{
		{Aria: "/zakelijk/a"},
		{Aria: "/zakelijk/b"},
	}
	landing := ast.Symbol{
		Name: "Section", Kind: ast.KindComponent,
		File: "https://www.example.com/zakelijk", Language: "ts",
		PageTitle: "Zakelijk",
		Contents:  contents,
		Links:     links,
	}
	body := renderOne(t, landing, "browse")
	if strings.Contains(body, "@kind:empty-state") {
		t.Errorf("section page must NOT emit empty-state scenario:\n%s", body)
	}
}

// URL-path hint flips the heuristic ON even when content shape is
// modest. A /blog or /news index needs the empty-state assertion.
func TestFeatureTemplate_EmptyState_OnBlogPath(t *testing.T) {
	landing := ast.Symbol{
		Name: "Blog", Kind: ast.KindComponent,
		File: "https://www.example.com/blog", Language: "ts",
		PageTitle: "Blog",
		Contents:  []ast.ContentAnchor{{Tag: "h1", Text: "Latest posts"}},
	}
	body := renderOne(t, landing, "browse")
	if !strings.Contains(body, "@kind:empty-state") {
		t.Errorf("/blog path must emit empty-state scenario:\n%s", body)
	}
}

// Structural fallback: lots of h2s + lots of links also flips it on.
func TestFeatureTemplate_EmptyState_OnDenseListShape(t *testing.T) {
	contents := []ast.ContentAnchor{{Tag: "h1", Text: "Index"}}
	for i := 0; i < 7; i++ {
		contents = append(contents, ast.ContentAnchor{Tag: "h2", Text: "Item"})
	}
	links := make([]ast.LocatorAnchor, 0, 7)
	for i := 0; i < 7; i++ {
		links = append(links, ast.LocatorAnchor{Aria: "/x"})
	}
	landing := ast.Symbol{
		Name: "Index", Kind: ast.KindComponent,
		File: "https://www.example.com/index", Language: "ts",
		Contents: contents, Links: links,
	}
	body := renderOne(t, landing, "browse")
	if !strings.Contains(body, "@kind:empty-state") {
		t.Errorf("dense h2+links must emit empty-state scenario:\n%s", body)
	}
}

func renderOne(t *testing.T, landing ast.Symbol, jk string) string {
	t.Helper()
	it := plan.Item{
		Symbol: landing, Symbols: []ast.Symbol{landing},
		PageURL: landing.File, Template: plan.TmplPlaywrightFeature,
		OutPath: "tests/e2e/features/x.feature", JourneyKind: jk,
	}
	out, err := Render([]plan.Item{it}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return string(out[0].Content)
}
