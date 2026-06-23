package llm

import "strings"

// NormalizeBaseURL returns an OpenAI-compatible chat-completions base URL
// that is guaranteed to end in exactly one "/v1" suffix. Idempotent for
// both bare endpoints ("http://host:11434") and ones already terminated
// with "/v1". Trailing slashes are stripped.
//
// Consumers that previously did
//
//	cfg.OpenAIBaseURL = strings.TrimRight(in, "/") + "/v1"
//
// would double the suffix when `in` already ended in "/v1", producing
// ".../v1/v1/chat/completions" and 404s against Ollama / vLLM. Use this
// helper to avoid that.
func NormalizeBaseURL(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "/v1") {
		return s
	}
	return s + "/v1"
}
