package probe

import (
	"testing"
)

// v0.90 regression: browserPageToMindmap was leaving p.Anchors nil,
// which silently broke form/contact/auth journey detection on every
// browser-probed page (isFormPage needs a `submit`-tagged Anchor to
// fire). Once we run ExtractHTMLAnchors over DOMHTML, the rendered
// submit button reaches the heuristics.
func TestBrowserPageToMindmap_PopulatesAnchorsFromDOM(t *testing.T) {
	bp := browserPage{
		URL:      "https://example.com/contact",
		FinalURL: "https://example.com/contact",
		Title:    "Contact",
		HasForm:  true,
		DOMHTML: `<html><body>
<form action="/submit" method="post">
  <input type="text" name="email" required />
  <button type="submit" aria-label="Send message">Send</button>
</form>
</body></html>`,
		Forms: []browserForm{{
			Action: "/submit",
			Method: "post",
			Inputs: []browserInput{{Tag: "input", Type: "text", Name: "email", Required: true}},
		}},
	}

	p := browserPageToMindmap(bp)
	if p == nil {
		t.Fatal("browserPageToMindmap returned nil")
	}
	if len(p.Anchors) == 0 {
		t.Fatal("p.Anchors empty — Anchors-from-DOM extraction did not run")
	}
	var submitFound bool
	for _, a := range p.Anchors {
		if a.Tag == "submit" {
			submitFound = true
			break
		}
	}
	if !submitFound {
		t.Errorf("no submit-tagged anchor in %v — form journey detection will silently no-op", p.Anchors)
	}
}

// When DOMHTML is empty (rare — script bug or capture failure), the
// page should still come through with no Anchors but no panic.
func TestBrowserPageToMindmap_HandlesEmptyDOM(t *testing.T) {
	bp := browserPage{URL: "https://example.com/", FinalURL: "https://example.com/", DOMHTML: ""}
	p := browserPageToMindmap(bp)
	if p == nil {
		t.Fatal("browserPageToMindmap nil on empty DOM")
	}
	if len(p.Anchors) != 0 {
		t.Errorf("expected no anchors for empty DOM; got %d", len(p.Anchors))
	}
}
