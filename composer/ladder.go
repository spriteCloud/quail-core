package composer

import (
	"context"
	"strings"
)

// Ladder is a fallback chain of LLM models. Each rung is tried in
// order; the first to return a parseable response wins. Designed to
// keep cheap-and-fast models as the default while letting expensive
// models rescue the parse-failure tail.
//
// Typical configuration:
//   primary:   qwen3-coder-next:latest  (fast)
//   fallback:  gpt-oss:120b             (slower, better instruction following)
type Ladder struct {
	// Rungs is the ordered list of (model, client) pairs. Try [0]
	// first; on parse failure, try [1], etc.
	Rungs []Rung
}

// Rung is one step in the model ladder.
type Rung struct {
	Model  string
	Client Client
}

// Empty returns true when the ladder has no rungs configured.
func (l Ladder) Empty() bool { return len(l.Rungs) == 0 }

// First returns the first rung, or zero-value Rung when empty.
func (l Ladder) First() Rung {
	if l.Empty() {
		return Rung{}
	}
	return l.Rungs[0]
}

// ProposeWithLadder walks the model ladder and returns the first
// rung's scenarios that parse cleanly. Embeds the winning model id
// in the @model:<id> tag on each returned scenario.
func ProposeWithLadder(ctx context.Context, ladder Ladder, j Journey, n int, fb Feedback) ([]ExtraScenario, string, error) {
	return ProposeWithLadderAndCache(ctx, ladder, j, n, fb, Cache{})
}

// ProposeWithLadderAndCache is the cache-aware variant. When the
// cache has a hit for the (model, journey) tuple of the FIRST rung,
// the cached scenarios are returned without an LLM call. Cache
// misses fall through to the normal ladder walk and the result is
// written back. Set Cache.Dir = "" to disable.
func ProposeWithLadderAndCache(ctx context.Context, ladder Ladder, j Journey, n int, fb Feedback, cache Cache) ([]ExtraScenario, string, error) {
	if ladder.Empty() {
		return nil, "", nil
	}
	primary := ladder.Rungs[0]
	primaryKey := CacheKey(primary.Model, j)
	if hit, ok := cache.Get(primaryKey); ok && len(hit) > 0 {
		return hit, primary.Model, nil
	}
	var lastErr error
	for _, rung := range ladder.Rungs {
		scenarios, err := ProposeWithFeedback(ctx, rung.Client, j, n, fb)
		if err == nil && len(scenarios) > 0 {
			tagged := tagWithModel(scenarios, rung.Model)
			// Store under the PRIMARY rung's key — the cache is keyed
			// by the model the caller asked for, not the fallback that
			// rescued the request.
			_ = cache.Put(primaryKey, tagged)
			return tagged, rung.Model, nil
		}
		lastErr = err
	}
	return nil, "", lastErr
}

// tagWithModel appends `@model:<id>` to each scenario's Tags so
// consumers can grep by what produced what. Slashes / colons get
// sanitized into a Gherkin-safe tag.
func tagWithModel(in []ExtraScenario, model string) []ExtraScenario {
	tag := "@model:" + tagSafeModelID(model)
	out := make([]ExtraScenario, len(in))
	for i, s := range in {
		s.Tags = append(append([]string{}, s.Tags...), tag)
		out[i] = s
	}
	return out
}

// tagSafeModelID normalises a model identifier (e.g.
// "hf.co/spriteCloud/quail:latest") into a Gherkin-tag-safe form.
// Slashes / colons are dropped; remaining chars stay verbatim.
func tagSafeModelID(model string) string {
	r := strings.NewReplacer("/", "-", ":", "-", " ", "-")
	return r.Replace(model)
}
