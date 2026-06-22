package config

import (
	"testing"
	"time"
)

func TestFromEnvDefaults(t *testing.T) {
	for _, k := range []string{
		"QUAIL_GITHUB_TOKEN", "GITHUB_TOKEN", "GITHUB_REPOSITORY",
		"OPENAI_BASE_URL", "OPENAI_API_KEY", "QUAIL_MODEL",
		"QUAIL_LLM_TIMEOUT", "QUAIL_LLM_TOKEN_CAP",
		"QUAIL_HEAL_MODE", "QUAIL_ALLOW_DIFF_TO_LLM",
		"QUAIL_BRANCH_PREFIX", "QUAIL_PLAYWRIGHT_REPORT",
		"QUAIL_WORKDIR", "QUAIL_PR",
	} {
		t.Setenv(k, "")
	}
	c := FromEnv()
	if c.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Errorf("OpenAIBaseURL default: %q", c.OpenAIBaseURL)
	}
	if c.Model != "gpt-4o-mini" {
		t.Errorf("Model default: %q", c.Model)
	}
	if c.LLMTimeout != 60*time.Second {
		t.Errorf("LLMTimeout default (v0.48 bumped 20s→60s for slower local LLMs): %v", c.LLMTimeout)
	}
	if c.LLMTokenCap != 600 {
		t.Errorf("LLMTokenCap default: %d", c.LLMTokenCap)
	}
	if c.HealMode != HealOnFailure {
		t.Errorf("HealMode default: %q", c.HealMode)
	}
	if c.BranchPrefix != "quail" {
		t.Errorf("BranchPrefix default: %q", c.BranchPrefix)
	}
	if c.WorkDir != "." {
		t.Errorf("WorkDir default: %q", c.WorkDir)
	}
	if c.AllowDiffToLLM {
		t.Error("AllowDiffToLLM should default to false")
	}
}

func TestFromEnvOverrides(t *testing.T) {
	t.Setenv("QUAIL_GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_REPOSITORY", "acme/widget")
	t.Setenv("OPENAI_BASE_URL", "http://local:8000/v1")
	t.Setenv("OPENAI_API_KEY", "key")
	t.Setenv("QUAIL_MODEL", "qwen")
	t.Setenv("QUAIL_LLM_TIMEOUT", "45s")
	t.Setenv("QUAIL_LLM_TOKEN_CAP", "1200")
	t.Setenv("QUAIL_HEAL_MODE", "proactive")
	t.Setenv("QUAIL_ALLOW_DIFF_TO_LLM", "1")
	t.Setenv("QUAIL_BRANCH_PREFIX", "qa")
	t.Setenv("QUAIL_PLAYWRIGHT_REPORT", "out/pw.json")
	t.Setenv("QUAIL_WORKDIR", "/tmp/repo")
	t.Setenv("QUAIL_PR", "42")
	c := FromEnv()
	if c.GitHubToken != "tok" || c.Repo != "acme/widget" {
		t.Errorf("auth: %+v", c)
	}
	if c.OpenAIBaseURL != "http://local:8000/v1" || c.OpenAIAPIKey != "key" || c.Model != "qwen" {
		t.Errorf("llm: %+v", c)
	}
	if c.LLMTimeout != 45*time.Second || c.LLMTokenCap != 1200 {
		t.Errorf("limits: %+v", c)
	}
	if c.HealMode != HealProactive || !c.AllowDiffToLLM {
		t.Errorf("heal: %+v", c)
	}
	if c.BranchPrefix != "qa" || c.PlaywrightReport != "out/pw.json" || c.WorkDir != "/tmp/repo" {
		t.Errorf("misc: %+v", c)
	}
	if c.PRNumber != 42 {
		t.Errorf("PR: %d", c.PRNumber)
	}
}

func TestValidateHealMode(t *testing.T) {
	for _, m := range []HealMode{HealOnFailure, HealProactive, HealOff} {
		if err := (Config{HealMode: m}).Validate(); err != nil {
			t.Errorf("%s should be valid: %v", m, err)
		}
	}
	if err := (Config{HealMode: HealMode("bogus")}).Validate(); err == nil {
		t.Error("bogus heal mode should error")
	}
}

func TestSplitRepo(t *testing.T) {
	cases := []struct {
		in    string
		owner string
		name  string
		ok    bool
	}{
		{"acme/widget", "acme", "widget", true},
		{"", "", "", false},
		{"bad", "", "", false},
		{"/missing-owner", "", "", false},
		{"missing-name/", "", "", false},
	}
	for _, tc := range cases {
		o, n, ok := (Config{Repo: tc.in}).SplitRepo()
		if o != tc.owner || n != tc.name || ok != tc.ok {
			t.Errorf("SplitRepo(%q) = (%q,%q,%v); want (%q,%q,%v)",
				tc.in, o, n, ok, tc.owner, tc.name, tc.ok)
		}
	}
}
