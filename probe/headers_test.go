package probe

import (
	"errors"
	"net/http"
	"testing"
)

func TestApplyDefaultHeaders_SendsBrowserShape(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	applyDefaultHeaders(req)
	ua := req.Header.Get("User-Agent")
	if ua == "" || ua[:5] != "Mozil" {
		t.Errorf("User-Agent should be browser-shaped; got %q", ua)
	}
	for _, h := range []string{"Accept-Language", "Sec-Fetch-Dest", "Sec-Fetch-Mode", "X-Quail-Probe", "Sec-Ch-Ua", "Upgrade-Insecure-Requests"} {
		if req.Header.Get(h) == "" {
			t.Errorf("missing header %s", h)
		}
	}
}

func TestLooksLikeWAFRejection(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{`probe: fetch https://x: Get "https://x": stream error: stream ID 9; INTERNAL_ERROR; received from peer`, true},
		{`tls: handshake failure`, true},
		{`probe: https://x returned 403 Forbidden`, true},
		{`probe: https://x returned 429 Too Many Requests`, true},
		{`probe: https://x returned 503 Service Unavailable`, true},
		{`connection reset by peer`, true},
		{`probe: https://x returned 200 OK`, false},
		{`probe: parse "x": invalid URL`, false},
	}
	for _, c := range cases {
		got := looksLikeWAFRejection(errors.New(c.err))
		if got != c.want {
			t.Errorf("looksLikeWAFRejection(%q) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestParseBrowserMode(t *testing.T) {
	cases := map[string]BrowserMode{
		"":        BrowserAuto,
		"auto":    BrowserAuto,
		"always":  BrowserAlways,
		"ALWAYS":  BrowserAlways,
		"never":   BrowserNever,
		"garbage": BrowserAuto,
	}
	for in, want := range cases {
		if got := ParseBrowserMode(in); got != want {
			t.Errorf("ParseBrowserMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseBrowserMode_LegacyEnvOverride(t *testing.T) {
	t.Setenv("QUAIL_BROWSER_PROBE", "1")
	if got := ParseBrowserMode(""); got != BrowserAlways {
		t.Errorf("legacy env should force BrowserAlways; got %v", got)
	}
}
