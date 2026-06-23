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
