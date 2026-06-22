package composer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type seqClient struct {
	replies []string
	errs    []error
	call    int
}

func (c *seqClient) Chat(_ context.Context, _, _ string) (string, error) {
	defer func() { c.call++ }()
	if c.call >= len(c.replies) {
		return "", errors.New("seqClient: out of replies")
	}
	var err error
	if c.call < len(c.errs) {
		err = c.errs[c.call]
	}
	return c.replies[c.call], err
}

func TestLadder_FirstRungWinsWhenValid(t *testing.T) {
	good := `[{"name":"x","steps":[{"keyword":"Given","text":"I open the landing page"}]}]`
	primary := &seqClient{replies: []string{good, good}}
	fallback := &seqClient{replies: []string{"never called"}}
	ladder := Ladder{Rungs: []Rung{
		{Model: "qwen3-coder-next", Client: primary},
		{Model: "gpt-oss:120b", Client: fallback},
	}}
	out, win, err := ProposeWithLadder(context.Background(), ladder, Journey{}, 3, Feedback{})
	if err != nil {
		t.Fatal(err)
	}
	if win != "qwen3-coder-next" {
		t.Errorf("winning model = %q; want qwen3-coder-next", win)
	}
	if len(out) != 1 {
		t.Errorf("expected 1 scenario; got %d", len(out))
	}
	if fallback.call != 0 {
		t.Errorf("fallback should NOT have been called; got %d calls", fallback.call)
	}
	// @model:qwen3-coder-next tag should be on the scenario.
	found := false
	for _, tag := range out[0].Tags {
		if tag == "@model:qwen3-coder-next" {
			found = true
		}
	}
	if !found {
		t.Errorf("scenario should carry @model tag; got %+v", out[0].Tags)
	}
}

func TestLadder_FallbackOnFirstRungParseFailure(t *testing.T) {
	good := `[{"name":"x","steps":[{"keyword":"Given","text":"I open the landing page"}]}]`
	// First rung returns garbage twice (initial + retry), second rung succeeds.
	primary := &seqClient{replies: []string{"garbage", "still garbage"}}
	fallback := &seqClient{replies: []string{good}}
	ladder := Ladder{Rungs: []Rung{
		{Model: "broken", Client: primary},
		{Model: "rescuer", Client: fallback},
	}}
	out, win, err := ProposeWithLadder(context.Background(), ladder, Journey{}, 3, Feedback{})
	if err != nil {
		t.Fatal(err)
	}
	if win != "rescuer" {
		t.Errorf("winning model = %q; want rescuer", win)
	}
	if len(out) != 1 {
		t.Errorf("expected 1 scenario from fallback; got %d", len(out))
	}
}

func TestLadder_TagSafeModelID(t *testing.T) {
	cases := map[string]string{
		"qwen3-coder-next:latest":              "qwen3-coder-next-latest",
		"hf.co/spriteCloud/quail:latest":       "hf.co-spriteCloud-quail-latest",
		"gpt oss 120b":                         "gpt-oss-120b",
	}
	for in, want := range cases {
		if got := tagSafeModelID(in); got != want {
			t.Errorf("tagSafeModelID(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestLadder_EmptyReturnsNothing(t *testing.T) {
	out, win, err := ProposeWithLadder(context.Background(), Ladder{}, Journey{}, 3, Feedback{})
	if err != nil || out != nil || win != "" {
		t.Errorf("empty ladder should be no-op; got out=%v win=%q err=%v", out, win, err)
	}
}

func TestTagWithModel_AppendsTagWithoutMutatingInput(t *testing.T) {
	in := []ExtraScenario{{Name: "a", Tags: []string{"@orig"}}}
	out := tagWithModel(in, "m")
	if len(in[0].Tags) != 1 || in[0].Tags[0] != "@orig" {
		t.Errorf("input mutated: %+v", in[0].Tags)
	}
	if !strings.Contains(strings.Join(out[0].Tags, " "), "@model:m") {
		t.Errorf("output missing model tag: %+v", out[0].Tags)
	}
}
