package gh

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// stripStubMux is a minimal mux that accepts every git/pulls call the
// OpenPR happy-path makes and records the files received by the tree
// POST so the test can assert which paths survived the strip.
func stripStubMux(t *testing.T, capturedFiles *map[string]string) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/widget/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ref": "refs/heads/main", "object": map[string]any{"sha": "basesha"},
		})
	})
	mux.HandleFunc("/repos/acme/widget/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sha": "blobsha"})
	})
	mux.HandleFunc("/repos/acme/widget/git/trees", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tree []struct {
				Path string `json:"path"`
			} `json:"tree"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, e := range body.Tree {
			(*capturedFiles)[e.Path] = ""
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sha": "treesha"})
	})
	mux.HandleFunc("/repos/acme/widget/git/commits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sha": "commitsha"})
	})
	mux.HandleFunc("/repos/acme/widget/git/refs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ref": "refs/heads/x"})
	})
	mux.HandleFunc("/repos/acme/widget/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"html_url": "https://github.com/acme/widget/pull/1",
		})
	})
	return mux
}

// TestOpenPR_StripsWorkflows_DefaultGHA is the legacy contract: under
// GITHUB_ACTIONS=true and without QUAIL_KEEP_WORKFLOWS, .github/workflows/*
// files are dropped from the push so the remote doesn't reject the
// whole push for missing `workflow` scope.
func TestOpenPR_StripsWorkflows_DefaultGHA(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("QUAIL_KEEP_WORKFLOWS", "")
	captured := map[string]string{}
	c, _ := newTestClient(t, stripStubMux(t, &captured))
	if _, err := c.OpenPR(context.Background(), PROpts{
		BaseBranch: "main", NewBranch: "x", Title: "t", Body: "b",
		Files: map[string][]byte{
			"tests/e2e/example.spec.ts":   []byte("// spec"),
			".github/workflows/e2e.yml":   []byte("# workflow"),
			".github/workflows/other.yml": []byte("# workflow"),
		},
	}); err != nil {
		t.Fatalf("OpenPR: %v", err)
	}
	if _, hit := captured[".github/workflows/e2e.yml"]; hit {
		t.Error("workflow file e2e.yml should have been stripped")
	}
	if _, hit := captured[".github/workflows/other.yml"]; hit {
		t.Error("workflow file other.yml should have been stripped")
	}
	if _, hit := captured["tests/e2e/example.spec.ts"]; !hit {
		t.Error("non-workflow files should NOT be stripped")
	}
}

// TestOpenPR_KeepsWorkflows_WhenEnvSet is the new opt-out: caller
// promises the token has `workflow` scope; binary leaves everything
// in the push set.
func TestOpenPR_KeepsWorkflows_WhenEnvSet(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("QUAIL_KEEP_WORKFLOWS", "1")
	captured := map[string]string{}
	c, _ := newTestClient(t, stripStubMux(t, &captured))
	if _, err := c.OpenPR(context.Background(), PROpts{
		BaseBranch: "main", NewBranch: "x", Title: "t", Body: "b",
		Files: map[string][]byte{
			"tests/e2e/example.spec.ts": []byte("// spec"),
			".github/workflows/e2e.yml": []byte("# workflow"),
		},
	}); err != nil {
		t.Fatalf("OpenPR: %v", err)
	}
	if _, hit := captured[".github/workflows/e2e.yml"]; !hit {
		t.Error("workflow file should be preserved when QUAIL_KEEP_WORKFLOWS=1")
	}
	if _, hit := captured["tests/e2e/example.spec.ts"]; !hit {
		t.Error("non-workflow file should also be preserved")
	}
	// Spot-check: every captured path is non-empty (no malformed key).
	for path := range captured {
		if strings.TrimSpace(path) == "" {
			t.Error("empty path captured")
		}
	}
}
