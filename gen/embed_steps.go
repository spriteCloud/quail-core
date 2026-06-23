package gen

import _ "embed"

// stepsBDDTemplate is the raw source of pw_steps_bdd.tmpl, embedded
// for callers (composer in particular) that need the list of
// registered step patterns without going through the full render
// pipeline. The step-def lines are pattern-literal (no Go template
// variables), so the raw template body is suitable input for
// composer.ExtractStepPatterns().
//
//go:embed templates/ts/pw_steps_bdd.tmpl
var stepsBDDTemplate []byte

// StepsBDDTemplate returns the raw bytes of pw_steps_bdd.tmpl. The
// result is the single source of truth for the composer's prompt
// "Registered step patterns" block: future template additions reach
// the LLM automatically once this list is extracted at compose time.
func StepsBDDTemplate() []byte { return stepsBDDTemplate }
