package composer

import (
	"context"
	"errors"
	"testing"
)

type fakeClient struct {
	reply string
	err   error
}

func (f fakeClient) Chat(ctx context.Context, system, user string) (string, error) {
	return f.reply, f.err
}

func TestPropose_HappyPath(t *testing.T) {
	json := `[
		{
			"name": "convert journey across multiple form fields",
			"tags": ["@kind:variant"],
			"steps": [
				{"keyword": "Given", "text": "I open the landing page"},
				{"keyword": "When", "text": "I enter \"jane@example.test\" into the \"email\" field"},
				{"keyword": "And", "text": "I submit the form"},
				{"keyword": "Then", "text": "no error message is shown in the form region"}
			]
		}
	]`
	out, err := Propose(context.Background(), fakeClient{reply: json}, Journey{Kind: "convert"}, 3)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(out) != 1 || out[0].Name == "" || len(out[0].Steps) != 4 {
		t.Fatalf("unexpected scenarios: %+v", out)
	}
}

func TestPropose_DropsScenariosWithUnregisteredSteps(t *testing.T) {
	json := `[
		{"name": "good", "steps": [
			{"keyword": "Given", "text": "I open the landing page"},
			{"keyword": "Then", "text": "no success message is shown"}
		]},
		{"name": "bad — invents a step", "steps": [
			{"keyword": "When", "text": "I do something completely off-script"}
		]}
	]`
	out, err := Propose(context.Background(), fakeClient{reply: json}, Journey{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "good" {
		t.Errorf("expected only the valid scenario; got %+v", out)
	}
}

func TestPropose_HandlesFencedResponse(t *testing.T) {
	// Some models wrap JSON in markdown fences; Parse strips them.
	reply := "Sure! Here you go:\n\n```json\n[{\"name\":\"x\",\"steps\":[{\"keyword\":\"Given\",\"text\":\"I open the landing page\"}]}]\n```\n"
	out, err := Propose(context.Background(), fakeClient{reply: reply}, Journey{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Errorf("expected 1 scenario after stripping fences; got %d", len(out))
	}
}

func TestPropose_LLMErrorBubbles(t *testing.T) {
	_, err := Propose(context.Background(), fakeClient{err: errors.New("boom")}, Journey{}, 3)
	if err == nil {
		t.Error("expected error when LLM returns one")
	}
}

func TestPropose_NilClientReturnsNothing(t *testing.T) {
	out, err := Propose(context.Background(), nil, Journey{}, 3)
	if err != nil || out != nil {
		t.Errorf("expected (nil, nil); got (%+v, %v)", out, err)
	}
}

func TestValidate_DropsEmptyNamesAndZeroSteps(t *testing.T) {
	in := []ExtraScenario{
		{Name: "", Steps: []Step{{Keyword: "Given", Text: "I open the landing page"}}},
		{Name: "no steps"},
		{Name: "good", Steps: []Step{{Keyword: "Given", Text: "I open the landing page"}}},
	}
	out := Validate(in)
	if len(out) != 1 || out[0].Name != "good" {
		t.Errorf("expected only the valid scenario; got %+v", out)
	}
}

func TestMatchesRegisteredPattern_ExactList(t *testing.T) {
	yes := []string{
		`I open the landing page`,
		`I open the page "/contact"`,
		`I click the link to "/about"`,
		`I navigate directly to "/login"`,
		`I enter "test@example.com" into the "email" field`,
		`I submit the form`,
		`I submit the form without filling any required field`,
		`the page title contains "Home"`,
		`the main heading reads "Welcome"`,
		`I see the heading "Contact"`,
		`no error message is shown in the form region`,
		`I remain on the same page`,
		`no success message is shown`,
	}
	for _, s := range yes {
		if !matchesRegisteredPattern(s) {
			t.Errorf("%q should match a registered pattern", s)
		}
	}
	no := []string{
		`I do something else entirely`,
		`Given I open the landing page`, // includes keyword — wrong; keyword is separate
		``,
		`I open the page`, // missing quoted arg
	}
	for _, s := range no {
		if matchesRegisteredPattern(s) {
			t.Errorf("%q should NOT match", s)
		}
	}
}
