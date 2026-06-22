// Package merge appends a generated test file into an existing one on a
// per-language basis. The goal is preserving existing tests rather than
// overwriting the file when the planner picks the idiomatic test path.
//
// Each language returns ok=false when it cannot safely merge; the caller is
// expected to fall back to a sibling filename in that case.
package merge

// Append returns the merged content of an existing test file and a freshly
// generated scaffold. ok=false means the language is unsupported or the
// merge would be unsafe (parse error, conflicting package, etc.).
func Append(language string, existing, generated []byte) (merged []byte, ok bool) {
	switch language {
	case "go":
		return appendGo(existing, generated)
	case "ts":
		return appendTS(existing, generated)
	case "python":
		return appendPython(existing, generated)
	}
	return nil, false
}
