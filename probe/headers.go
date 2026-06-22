package probe

import (
	"net/http"
	"strings"
)

// Chrome-shaped UA. WAFs (Akamai, Cloudflare Bot Manager, etc.)
// reject obvious-bot UAs by default. v0.86 swaps the long-lived
// `quail-probe/1` UA for a recent Chrome-on-Linux string — same
// shape every modern browser sends. We still send the real UA via
// the `X-Quail-Probe` header so origins that *want* to identify
// us can.
const chromeUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

// quailIdentHeader lets honest sites identify the probe without
// us giving the game away to bot managers. Servers can choose to
// allow us based on this header; the UA stays browser-like.
const quailIdentHeader = "X-Quail-Probe"
const quailIdentValue = "quail-probe/1 (+https://github.com/spriteCloud/quail)"

// applyDefaultHeaders sets the full set of headers a modern Chrome
// sends on a top-level navigation. Used by Fetch + the contract /
// graphql / webhook probes so every outbound request looks the
// same. Callers can override individual headers after this call.
func applyDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", chromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// We deliberately don't advertise br/zstd — net/http doesn't
	// transparently decode those, and we'd get garbled bytes back.
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Sec-Ch-Ua", `"Not?A_Brand";v="99", "Chromium";v="130", "Google Chrome";v="130"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set(quailIdentHeader, quailIdentValue)
}

// looksLikeWAFRejection reports whether an error from the static
// HTTP probe matches a pattern that suggests the origin's WAF /
// bot manager rejected us, rather than the site genuinely being
// unreachable. The caller uses this to decide whether to retry
// the URL through the Playwright browser probe (which presents as
// a real Chromium and almost always gets through).
//
// Patterns matched:
//   - HTTP/2 "INTERNAL_ERROR" / "REFUSED_STREAM" / "CANCEL" (the
//     Akamai/Cloudflare drop signature)
//   - TLS handshake failures (banks frequently send TCP RST mid-
//     handshake to obvious-bot fingerprints)
//   - 403 / 406 / 429 (the typical body-less block responses)
//   - 503 with a Cloudflare/Akamai server header (we don't have
//     the response here; the error message includes the status so
//     a substring check is enough)
//   - generic "connection reset by peer" mid-request
// LooksLikeWAFRejection is exported for the CLI's finishProbe so
// it can decide whether to suggest --browser=always in its error
// message. Same behaviour as the internal looksLikeWAFRejection.
func LooksLikeWAFRejection(err error) bool { return looksLikeWAFRejection(err) }

func looksLikeWAFRejection(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	signatures := []string{
		"internal_error",       // HTTP/2 stream rejection
		"refused_stream",       // HTTP/2 stream rejection
		"http2: stream cancel", // HTTP/2 stream cancel
		"tls handshake",
		"tls: handshake failure",
		"connection reset by peer",
		"eof",
		"returned 403",
		"returned 406",
		"returned 429",
		"returned 503",
		"forbidden",
	}
	for _, s := range signatures {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
