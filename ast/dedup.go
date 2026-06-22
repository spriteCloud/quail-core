package ast

// DedupAnchors collapses adjacent / repeated anchors that target the same
// element via the same locator hint. Used by both the TSX extractor and the
// HTML/probe path so generated specs never carry visibility duplicates
// like three consecutive `getByRole('banner').toBeVisible()` calls.
//
// The dedup key is the tuple (TestID, Role, Aria, Tag, Name). Empty fields
// participate — two anchors with no testid but distinct roles are kept.
//
// Output is capped at maxAnchors to keep specs short. Tag-shaped link
// anchors ("link-a", "link-to") pass through untouched; they're routed
// elsewhere by the caller.
const maxAnchors = 8

func DedupAnchors(in []LocatorAnchor) []LocatorAnchor {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]LocatorAnchor, 0, len(in))
	visibilityCount := 0
	for _, a := range in {
		if a.Tag == "link-a" || a.Tag == "link-to" {
			out = append(out, a)
			continue
		}
		key := a.TestID + "|" + a.Role + "|" + a.Aria + "|" + a.Tag + "|" + a.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		visibilityCount++
		if visibilityCount > maxAnchors {
			continue
		}
		out = append(out, a)
	}
	return out
}

// DedupInputs collapses duplicate form inputs by (Name, Type, TestID).
func DedupInputs(in []FormInput) []FormInput {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]FormInput, 0, len(in))
	for _, i := range in {
		key := i.TestID + "|" + i.Name + "|" + i.Type
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, i)
	}
	return out
}

// DedupLinks collapses anchors carrying the same href (stored in Aria
// during extraction). When multiple anchors share an href, the one with
// non-empty Text wins — otherwise an image-link with empty text would
// suppress the textual nav link and downstream consumers would attribute
// the wrong label to the URL.
func DedupLinks(in []LocatorAnchor) []LocatorAnchor {
	if len(in) == 0 {
		return nil
	}
	indexByHref := map[string]int{}
	out := make([]LocatorAnchor, 0, len(in))
	for _, l := range in {
		if idx, ok := indexByHref[l.Aria]; ok {
			// Prefer the entry with non-empty Text.
			if out[idx].Text == "" && l.Text != "" {
				out[idx] = l
			}
			continue
		}
		indexByHref[l.Aria] = len(out)
		out = append(out, l)
	}
	return out
}
