package llm

import "testing"

// v0.8.1 contract: structurePreserved tolerates LLM reformatting on
// test-keyword lines as long as the keyword COUNT is unchanged.

func TestStructurePreserved_AcceptsArgRemovalOnTestLine(t *testing.T) {
	before := []byte(`import { test } from '@playwright/test'
test('Spritecloud foo', { timeout: 5000 }, async ({ page }) => {
  await page.goto('/')
})
`)
	after := []byte(`import { test } from '@playwright/test'
test('Verify the foo journey', async ({ page }) => {
  await page.goto('/')
})
`)
	if !structurePreserved("ts", before, after) {
		t.Error("structure should be preserved — same 1 import + 1 test keyword")
	}
}

func TestStructurePreserved_RejectsAddedTest(t *testing.T) {
	before := []byte(`import { test } from '@playwright/test'
test('foo', async ({ page }) => {})
`)
	after := []byte(`import { test } from '@playwright/test'
test('foo', async ({ page }) => {})
test('extra', async ({ page }) => {})
`)
	if structurePreserved("ts", before, after) {
		t.Error("structure should NOT be preserved — extra test added")
	}
}

func TestStructurePreserved_RejectsDroppedDescribe(t *testing.T) {
	before := []byte(`describe('outer', () => {
  it('inner', async () => {})
})
`)
	after := []byte(`it('inner', async () => {})
`)
	if structurePreserved("ts", before, after) {
		t.Error("structure should NOT be preserved — describe removed")
	}
}

func TestStructurePreserved_RejectsAddedImport(t *testing.T) {
	before := []byte(`import { test } from '@playwright/test'
test('foo', async ({ page }) => {})
`)
	after := []byte(`import { test } from '@playwright/test'
import fs from 'fs'
test('foo', async ({ page }) => {})
`)
	if structurePreserved("ts", before, after) {
		t.Error("structure should NOT be preserved — import added")
	}
}

func TestStructurePreserved_AcceptsTitleAndCommentChanges(t *testing.T) {
	before := []byte(`// header
import { describe, it } from 'vitest'
describe('foo', () => {
  it('bar', async () => {})
})
`)
	after := []byte(`// Verify the foo feature behaves
import { describe, it } from 'vitest'
describe('Verify foo coverage', () => {
  it('Verify bar behaviour', async () => {})
})
`)
	if !structurePreserved("ts", before, after) {
		t.Error("structure should be preserved — only titles + comment text changed")
	}
}

func TestStructurePreserved_RejectsEmpty(t *testing.T) {
	if structurePreserved("ts", []byte("test('x', async () => {})\n"), nil) {
		t.Error("empty after should reject")
	}
}

// v0.8.3 — actual qwen3-coder-next corruption observed against
// spritecloud.com: backticks dropped from a template-literal title.
// Keyword counts still match; the old loosen-too-much check accepted
// this and the suite failed to parse with `Unexpected token`.
func TestStructurePreserved_RejectsDroppedBacktick(t *testing.T) {
	before := []byte(`import { test } from '@playwright/test'
for (const zoom of [2.0, 4.0]) {
  test(` + "`handles ${zoom * 100}% zoom`" + `, async ({ page }) => {})
}
`)
	after := []byte(`import { test } from '@playwright/test'
for (const zoom of [2.0, 4.0]) {
  test(handles ${zoom * 100}% zoom, async ({ page }) => {})
}
`)
	if structurePreserved("ts", before, after) {
		t.Error("dropped backticks around template-literal title should fail syntax sanity")
	}
}

// v0.8.3 — observed against spritecloud.com: LLM replaced an
// `expect(body).not.toContain(...)` line with bare English prose.
// Keyword counts still match. This must fail.
func TestStructurePreserved_RejectsBareEnglishStatement(t *testing.T) {
	before := []byte(`import { test, expect } from '@playwright/test'
test('foo', async ({ page }) => {
  const body = ''
  expect(body).not.toContain('Traceback')
  expect(body).not.toContain('at Object')
})
`)
	after := []byte(`import { test, expect } from '@playwright/test'
test('foo', async ({ page }) => {
  const body = ''
  No server error traceback in body
  No stack trace in body
})
`)
	if structurePreserved("ts", before, after) {
		t.Error("bare-English replacement of an expect line should fail syntax sanity")
	}
}

// Inverse — the LLM's well-formed humanize-only rewrite must still
// pass. Keyword counts unchanged, every test-call has a quoted first
// arg, quote balance preserved.
func TestStructurePreserved_AcceptsWellFormedHumanize(t *testing.T) {
	before := []byte(`import { test, expect } from '@playwright/test'
test('Spritecloud foo', async ({ page }) => {
  await page.goto('/')
})
`)
	after := []byte(`import { test, expect } from '@playwright/test'
test('Verify the foo journey', async ({ page }) => {
  await page.goto('/')
})
`)
	if !structurePreserved("ts", before, after) {
		t.Error("well-formed title rewrite should pass")
	}
}
