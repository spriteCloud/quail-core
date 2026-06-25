package composer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeClient lets tests inject a canned response without spinning up
// an LLM. Mirrors composer_test.go's fakeClient.
type classifyFakeClient struct {
	resp string
	err  error
}

func (f classifyFakeClient) Chat(ctx context.Context, system, user string) (string, error) {
	return f.resp, f.err
}

func TestClassifyPageIntent_HappyPath(t *testing.T) {
	c := classifyFakeClient{resp: `{"intent":"calculator","confidence":0.92,"vertical":"mortgage","key_assertions":["max loan goes up with income"]}`}
	got, err := ClassifyPageIntent(context.Background(), c, PageInput{
		URL: "https://ing.nl/hypotheek", Title: "Hypotheek berekenen", H1: "Hypotheek berekenen",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "calculator" {
		t.Errorf("intent = %q, want calculator", got.Intent)
	}
	if got.Confidence < 0.9 {
		t.Errorf("confidence = %f, want >= 0.9", got.Confidence)
	}
	if got.Vertical != "mortgage" {
		t.Errorf("vertical = %q, want mortgage", got.Vertical)
	}
	if len(got.KeyAssertions) != 1 {
		t.Errorf("key_assertions len = %d, want 1", len(got.KeyAssertions))
	}
}

func TestClassifyPageIntent_NilClient_ReturnsEmpty(t *testing.T) {
	got, err := ClassifyPageIntent(context.Background(), nil, PageInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "" {
		t.Errorf("nil client must return zero intent; got %q", got.Intent)
	}
}

func TestClassifyPageIntent_ErrorPropagates(t *testing.T) {
	c := classifyFakeClient{err: errors.New("boom")}
	_, err := ClassifyPageIntent(context.Background(), c, PageInput{})
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

// Unknown intents (LLM invented a new one) must be normalised to
// empty so templates fall back to generic. Confidence is also wiped
// — a confidence on an unknown intent is meaningless.
func TestParseIntent_UnknownIntentNormalisedToEmpty(t *testing.T) {
	got, err := ParseIntent(`{"intent":"frobnicator","confidence":0.95}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "" {
		t.Errorf("unknown intent must be blanked; got %q", got.Intent)
	}
	if got.Confidence != 0 {
		t.Errorf("unknown intent must reset confidence; got %f", got.Confidence)
	}
}

// Models sometimes wrap JSON in ```json fences despite the prompt.
// ParseIntent must tolerate that.
func TestParseIntent_StripsMarkdownFences(t *testing.T) {
	got, err := ParseIntent("```json\n{\"intent\":\"booking\",\"confidence\":0.8}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "booking" {
		t.Errorf("intent = %q, want booking", got.Intent)
	}
}

// Models sometimes add a one-line preamble before the JSON. Slice
// from { to } must survive that.
func TestParseIntent_TolerantToPreamble(t *testing.T) {
	got, err := ParseIntent(`Here is the classification: {"intent":"signup","confidence":0.71}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "signup" {
		t.Errorf("intent = %q, want signup", got.Intent)
	}
}

func TestParseIntent_ConfidenceClampedToUnit(t *testing.T) {
	got, _ := ParseIntent(`{"intent":"calculator","confidence":1.7}`)
	if got.Confidence != 1 {
		t.Errorf("confidence > 1 must clamp to 1; got %f", got.Confidence)
	}
	got, _ = ParseIntent(`{"intent":"calculator","confidence":-0.3}`)
	if got.Confidence != 0 {
		t.Errorf("negative confidence must clamp to 0; got %f", got.Confidence)
	}
}

func TestParseIntent_NullVerticalBlanked(t *testing.T) {
	got, _ := ParseIntent(`{"intent":"calculator","confidence":0.8,"vertical":"null"}`)
	if got.Vertical != "" {
		t.Errorf("'null' vertical must blank; got %q", got.Vertical)
	}
}

// IntentFingerprint excludes URL — two pages with identical
// title/H1/meta hit the cache regardless of URL. Saves classifier
// calls across /us/ vs /uk/ variants.
func TestIntentFingerprint_ExcludesURL(t *testing.T) {
	a := IntentFingerprint(PageInput{URL: "https://x.test/us/a", Title: "T", H1: "H"})
	b := IntentFingerprint(PageInput{URL: "https://x.test/uk/a", Title: "T", H1: "H"})
	if a != b {
		t.Errorf("fingerprint must ignore URL; got\nA=%s\nB=%s", a, b)
	}
}

func TestIntentFingerprint_ChangesOnTitleDiff(t *testing.T) {
	a := IntentFingerprint(PageInput{Title: "A"})
	b := IntentFingerprint(PageInput{Title: "B"})
	if a == b {
		t.Error("fingerprint must change when Title changes")
	}
}

// ClassifyWithLadderAndCache: cache hit returns instantly without
// invoking the LLM. The fake client below would error if called.
func TestClassifyWithLadderAndCache_HitsCache(t *testing.T) {
	tmp := t.TempDir()
	cache := Cache{Dir: tmp}
	model := "test-model"
	p := PageInput{URL: "https://x.test/", Title: "Calc"}
	cached := PageIntent{Intent: "calculator", Confidence: 0.9}
	if err := cache.PutIntent(IntentCacheKey(model, p), cached); err != nil {
		t.Fatal(err)
	}
	ladder := Ladder{Rungs: []Rung{{Model: model, Client: classifyFakeClient{err: errors.New("must not be called")}}}}
	got, gotModel, err := ClassifyWithLadderAndCache(context.Background(), ladder, p, cache)
	if err != nil {
		t.Fatalf("cache hit must not error; got %v", err)
	}
	if got.Intent != "calculator" {
		t.Errorf("expected cached intent; got %q", got.Intent)
	}
	if gotModel != model {
		t.Errorf("model = %q, want %q", gotModel, model)
	}
}

// Empty ladder is a no-op (matches Propose behaviour).
func TestClassifyWithLadderAndCache_EmptyLadder(t *testing.T) {
	got, _, err := ClassifyWithLadderAndCache(context.Background(), Ladder{}, PageInput{}, Cache{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "" {
		t.Errorf("empty ladder must return zero intent; got %q", got.Intent)
	}
}

// Primary rung returns empty (low-confidence/unknown) → ladder falls
// through to the secondary rung. Mirrors composer_test ladder fallback.
func TestClassifyWithLadderAndCache_FallbackToSecondary(t *testing.T) {
	primary := classifyFakeClient{resp: `{"intent":"","confidence":0}`}
	secondary := classifyFakeClient{resp: `{"intent":"booking","confidence":0.85}`}
	ladder := Ladder{Rungs: []Rung{
		{Model: "primary", Client: primary},
		{Model: "secondary", Client: secondary},
	}}
	got, model, err := ClassifyWithLadderAndCache(context.Background(), ladder, PageInput{Title: "Book"}, Cache{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != "booking" {
		t.Errorf("expected fallback to secondary; got %q", got.Intent)
	}
	if model != "secondary" {
		t.Errorf("model = %q, want secondary", model)
	}
}

// Prompt must include the page metadata fields. Smoke test that the
// classifier surfaces enough signal for the model to reason on.
func TestBuildClassifyUserPrompt_IncludesAllFields(t *testing.T) {
	p := PageInput{
		URL: "https://x.test/calc", Title: "Hypotheek berekenen", H1: "Bereken je hypotheek",
		MetaDescription: "Wat kun je lenen?", OGType: "website",
		FormSummary: "3 inputs: applicant (number), partner (number), monthly (number)",
		Labels:      []string{"Jouw bruto jaarinkomen", "Bruto jaarinkomen partner"},
	}
	prompt := buildClassifyUserPrompt(p)
	for _, want := range []string{
		"https://x.test/calc", "Hypotheek berekenen", "Bereken je hypotheek",
		"Wat kun je lenen?", "website", "applicant (number)", "Jouw bruto jaarinkomen",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\n%s", want, prompt)
		}
	}
}
