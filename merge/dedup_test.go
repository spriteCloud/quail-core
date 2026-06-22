package merge

import (
	"strings"
	"testing"
)

func TestAppendTSDedupesCaseDifferentDescribes(t *testing.T) {
	existing := []byte(tsHeader + `import { test, expect } from '@playwright/test'

test.describe.configure({ mode: 'parallel' })
test.describe('spriteCloud — race @ https://www.spritecloud.com/', () => {
  test('@kind:race @smoke form survives a rapid double-submit', async ({ page }) => {
    await page.goto('/')
  })
})
`)
	// Same content, different casing for the symbol prefix.
	generated := []byte(tsHeader + `import { test, expect } from '@playwright/test'

test.describe.configure({ mode: 'parallel' })
test.describe('Spritecloud — race @ https://www.spritecloud.com/', () => {
  test('@kind:race @smoke form survives a rapid double-submit', async ({ page }) => {
    await page.goto('/')
  })
})
`)
	out, _ := appendTS(existing, generated)
	count := strings.Count(string(out), "test.describe('")
	if count != 1 {
		t.Errorf("expected 1 describe after dedup, got %d in:\n%s", count, out)
	}
}

func TestAppendTSKeepsDistinctDescribes(t *testing.T) {
	existing := []byte(tsHeader + `import { test, expect } from '@playwright/test'

test.describe('spriteCloud — race @ https://www.spritecloud.com/', () => {
  test('a', async () => {})
})
`)
	// Different kind tag, different content — must be kept.
	generated := []byte(tsHeader + `import { test, expect } from '@playwright/test'

test.describe('spriteCloud — perf @ https://www.spritecloud.com/', () => {
  test('b', async () => {})
})
`)
	out, _ := appendTS(existing, generated)
	if !strings.Contains(string(out), "race @") || !strings.Contains(string(out), "perf @") {
		t.Errorf("both distinct describes should be present, got:\n%s", out)
	}
}
