package composer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PageIntent is the LLM-derived domain classification for a probed
// page. Mirror of ast.PageIntent (kept here to avoid composer → ast
// import; the wiring layer converts). Zero value means "unknown".
type PageIntent struct {
	Intent        string   `json:"intent"`
	Confidence    float64  `json:"confidence"`
	Vertical      string   `json:"vertical,omitempty"`
	KeyAssertions []string `json:"key_assertions,omitempty"`
}

// PageInput is the probe-side context the classifier renders into the
// model prompt. Same shape philosophy as Journey: all fields optional;
// model handles partial data; missing structure → empty intent.
type PageInput struct {
	URL             string
	Title           string
	H1              string
	MetaDescription string
	OGType          string
	Labels          []string // visible labels from PrimaryComponent (top ~12)
	FormSummary     string   // one-line digest of form shape, e.g. "3 inputs: bruto_jaarinkomen (number), partner_inkomen (number), bruto_maandbedrag (number)"
}

// allowedIntents bounds the classifier's output to the set the
// templates know about. Anything outside this set is normalized to
// empty (== "fall back to generic").
var allowedIntents = map[string]bool{
	"calculator":   true,
	"booking":      true,
	"checkout":     true,
	"signup":       true,
	"search":       true,
	"quote":        true,
	"configurator": true,
	"marketing":    true,
	"content":      true,
}

const classifySystemPrompt = `You are a senior QA engineer classifying a single web page so a test generator can pick the right scenario shapes.

Read the page metadata and pick ONE intent from this exact list:
- calculator     (mortgage / loan / insurance / tax / pension widget that computes a number from inputs)
- booking        (travel / restaurant / appointment with check-in/out or date pair)
- checkout       (e-commerce cart / order summary with line items, totals, promo, payment)
- signup         (account creation: email + password + confirm)
- search         (free-text query → results listing)
- quote          (multi-step form that ends in a price estimate; calculator-shaped but multi-page)
- configurator   (build-your-X: a car, a pizza, a plan — selections drive a running total or summary)
- marketing      (landing / hero / section page; CTAs but no submit-to-result widget)
- content        (article / blog post / documentation page — long-form prose)

Return ONLY a JSON object on a single line, no markdown, no commentary:
{"intent": "<one of above>", "confidence": <0.0-1.0>, "vertical": "<mortgage|insurance|travel|ecommerce|saas|gov|null>", "key_assertions": ["...", "..."]}

Rules:
- confidence is your honest belief. 0.9 = certain, 0.7 = leaning, 0.5 = guessing, 0.3 = stab in the dark.
- vertical is the business domain. Use null when unclear.
- key_assertions are 1-3 short English sentences naming what a test would check on THIS specific page. Be concrete ("the maximum loan amount goes up when annual income goes up"). Skip if you have nothing specific.
- If you cannot identify any of the listed intents, return {"intent": "", "confidence": 0.0}. Do not invent a new intent.`

// ClassifyPageIntent runs one LLM call to classify the page. Returns
// zero PageIntent (not an error) when llm is nil so callers can
// always invoke this without an availability check.
func ClassifyPageIntent(ctx context.Context, llm Client, p PageInput) (PageIntent, error) {
	if llm == nil {
		return PageIntent{}, nil
	}
	user := buildClassifyUserPrompt(p)
	raw, err := llm.Chat(ctx, classifySystemPrompt, user)
	if err != nil {
		return PageIntent{}, fmt.Errorf("classify: llm chat: %w", err)
	}
	return ParseIntent(raw)
}

// ParseIntent extracts the JSON object from the model's raw response
// and normalizes it. Out-of-vocabulary intents are blanked. Exposed
// for unit tests; production callers use ClassifyPageIntent.
func ParseIntent(raw string) (PageIntent, error) {
	s := strings.TrimSpace(raw)
	// Strip markdown fences if the model added them despite the prompt.
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	// Slice from the first { to the last } to tolerate prelude/postlude.
	if i := strings.Index(s, "{"); i > 0 {
		s = s[i:]
	}
	if j := strings.LastIndex(s, "}"); j >= 0 && j < len(s)-1 {
		s = s[:j+1]
	}
	var out PageIntent
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return PageIntent{}, fmt.Errorf("classify: parse json: %w (raw=%q)", err, raw)
	}
	// Normalize: lowercase + drop unknown intents.
	out.Intent = strings.ToLower(strings.TrimSpace(out.Intent))
	if !allowedIntents[out.Intent] {
		out.Intent = ""
		out.Confidence = 0
	}
	out.Vertical = strings.ToLower(strings.TrimSpace(out.Vertical))
	if out.Vertical == "null" {
		out.Vertical = ""
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	return out, nil
}

func buildClassifyUserPrompt(p PageInput) string {
	var sb strings.Builder
	sb.WriteString("Classify this page:\n\n")
	if p.URL != "" {
		fmt.Fprintf(&sb, "URL: %s\n", p.URL)
	}
	if p.Title != "" {
		fmt.Fprintf(&sb, "<title>: %s\n", p.Title)
	}
	if p.H1 != "" {
		fmt.Fprintf(&sb, "<h1>: %s\n", p.H1)
	}
	if p.MetaDescription != "" {
		fmt.Fprintf(&sb, "meta description: %s\n", p.MetaDescription)
	}
	if p.OGType != "" {
		fmt.Fprintf(&sb, "og:type: %s\n", p.OGType)
	}
	if p.FormSummary != "" {
		fmt.Fprintf(&sb, "form: %s\n", p.FormSummary)
	}
	if len(p.Labels) > 0 {
		sb.WriteString("visible input labels: ")
		sb.WriteString(strings.Join(p.Labels, ", "))
		sb.WriteString("\n")
	}
	sb.WriteString("\nReturn the JSON object now.")
	return sb.String()
}

// IntentFingerprint is the cache key for a classified page. Same
// content → same fingerprint → cache hit. Excludes URL by design:
// two pages with identical title/H1/meta should produce the same
// intent regardless of URL (saves classifier calls across language
// variants, /us/ vs /uk/, etc.).
func IntentFingerprint(p PageInput) string {
	parts := []string{
		"t:" + p.Title,
		"h:" + p.H1,
		"d:" + p.MetaDescription,
		"og:" + p.OGType,
		"f:" + p.FormSummary,
		"l:" + strings.Join(p.Labels, "|"),
	}
	return strings.Join(parts, "\n")
}
