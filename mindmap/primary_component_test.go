package mindmap

import (
	"testing"

	"github.com/spriteCloud/quail-core/ast"
)

// A page that carries a PrimaryComponent must still tag-derive the
// same way it would without one — the new field is purely additive
// downstream signal, not a tag input. Mortgage-calculator shape:
// numeric inputs, a role=button submit, "berekenen" in the URL.
func TestPrimaryComponent_NoRegressionOnTagging(t *testing.T) {
	p := &Page{
		URL:     "https://anybank.example/hypotheek-berekenen",
		HasForm: true,
		Inputs: []ast.FormInput{
			{Name: "bruto", Type: "number"},
			{Name: "partner", Type: "number"},
			{Name: "energielabel", Type: "select", OptionValues: []string{"A", "B", "C"}},
		},
		Anchors: []ast.LocatorAnchor{{Role: "button"}},
		PrimaryComponent: &ast.PrimaryComponent{
			Selector: "flex-calc",
			Inputs: []ast.FormInput{
				{Name: "bruto", Type: "number"},
				{Name: "partner", Type: "number"},
				{Name: "energielabel", Type: "select", OptionValues: []string{"A", "B", "C"}},
			},
		},
	}
	tags := TagsFromPage(p)
	hasTag := func(t string) bool {
		for _, x := range tags {
			if x == t {
				return true
			}
		}
		return false
	}
	if !hasTag(TagForm) {
		t.Errorf("calculator-shape page must still emit %q tag, got %v", TagForm, tags)
	}
}

// Nil PrimaryComponent must round-trip — older crawl paths and the
// static (non-browser) probe never populate it, so the rest of the
// pipeline has to keep working.
func TestPrimaryComponent_NilStaysNil(t *testing.T) {
	p := &Page{URL: "https://example.com/", Title: "Home"}
	if p.PrimaryComponent != nil {
		t.Errorf("zero-value Page must have nil PrimaryComponent")
	}
	_ = TagsFromPage(p) // must not panic on nil ComponentRef
}
