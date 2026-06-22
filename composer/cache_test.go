package composer

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Cache{Dir: dir}
	want := []ExtraScenario{
		{Name: "x", Steps: []Step{{Keyword: "Given", Text: "I open the landing page"}}},
	}
	if err := c.Put("k1", want); err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get("k1")
	if !ok || len(got) != 1 || got[0].Name != "x" {
		t.Errorf("round-trip mismatch: ok=%v got=%+v", ok, got)
	}
	// File should exist on disk.
	if _, err := readFile(filepath.Join(dir, "k1.json")); err != nil {
		t.Errorf("cache file should be on disk: %v", err)
	}
}

func TestCache_MissReturnsFalse(t *testing.T) {
	c := Cache{Dir: t.TempDir()}
	got, ok := c.Get("never-stored")
	if ok || got != nil {
		t.Errorf("expected miss; got ok=%v got=%+v", ok, got)
	}
}

func TestCache_DisabledByEmptyDir(t *testing.T) {
	c := Cache{}
	if err := c.Put("k", nil); err == nil {
		t.Error("Put on disabled cache should return an error")
	}
	if _, ok := c.Get("k"); ok {
		t.Error("Get on disabled cache should be a miss")
	}
}

func TestCacheKey_StableAcrossEqualJourneys(t *testing.T) {
	a := Journey{URL: "https://x.test/", Kind: "convert", Title: "T", H1: "H", Links: []string{"/a", "/b"}}
	b := Journey{URL: "https://x.test/", Kind: "convert", Title: "T", H1: "H", Links: []string{"/a", "/b"}}
	if CacheKey("m", a) != CacheKey("m", b) {
		t.Error("equal journeys should hash to the same key")
	}
}

func TestCacheKey_DiffersWhenLinksChange(t *testing.T) {
	a := Journey{URL: "x", Links: []string{"/a"}}
	b := Journey{URL: "x", Links: []string{"/b"}}
	if CacheKey("m", a) == CacheKey("m", b) {
		t.Error("links change should change the key")
	}
}

func TestCacheKey_DiffersAcrossModels(t *testing.T) {
	j := Journey{URL: "x", Kind: "convert"}
	if CacheKey("m1", j) == CacheKey("m2", j) {
		t.Error("model change should change the key")
	}
}

func TestResolveCacheDir_Precedence(t *testing.T) {
	t.Setenv("QUAIL_LLM_CACHE", "")
	if got := ResolveCacheDir("", "/work"); got != "" {
		t.Errorf("no opt-in signal → empty; got %q", got)
	}
	t.Setenv("QUAIL_LLM_CACHE", "auto")
	if got := ResolveCacheDir("", "/work"); got != "/work/.quail-cache" {
		t.Errorf(`"auto" + workdir → /work/.quail-cache; got %q`, got)
	}
	t.Setenv("QUAIL_LLM_CACHE", "/explicit/path")
	if got := ResolveCacheDir("", "/work"); got != "/explicit/path" {
		t.Errorf("env path wins; got %q", got)
	}
	// Explicit arg overrides env.
	if got := ResolveCacheDir("/from-flag", "/work"); got != "/from-flag" {
		t.Errorf("explicit overrides env; got %q", got)
	}
}

func TestProposeWithLadderAndCache_HitsCacheSecondTime(t *testing.T) {
	good := `[{"name":"x","steps":[{"keyword":"Given","text":"I open the landing page"}]}]`
	client := &seqClient{replies: []string{good}}
	ladder := Ladder{Rungs: []Rung{{Model: "m", Client: client}}}
	cache := Cache{Dir: t.TempDir()}
	j := Journey{URL: "https://x.test/", Kind: "convert"}

	// First call hits the LLM and writes to cache.
	first, _, err := ProposeWithLadderAndCache(context.Background(), ladder, j, 3, Feedback{}, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 {
		t.Fatalf("first call should return 1 scenario; got %d", len(first))
	}
	if client.call != 1 {
		t.Errorf("first call should make 1 LLM call; got %d", client.call)
	}

	// Second call: same inputs → cache hit, no LLM call.
	second, _, err := ProposeWithLadderAndCache(context.Background(), ladder, j, 3, Feedback{}, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Errorf("second call should return cached scenario; got %d", len(second))
	}
	if client.call != 1 {
		t.Errorf("second call should NOT make an LLM call; got %d total", client.call)
	}
}

func readFile(p string) ([]byte, error) {
	// Tiny wrapper to keep imports minimal in the test.
	return readFileImpl(p)
}
