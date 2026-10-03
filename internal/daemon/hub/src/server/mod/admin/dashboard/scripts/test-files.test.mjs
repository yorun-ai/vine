import assert from 'node:assert/strict'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { discoverTests } from './test-files.mjs'

test('discovers tests in new feature directories and ignores fixtures and browser specs', (t) => {
  const root = mkdtempSync(join(tmpdir(), 'vine-dashboard-test-files-'))
  t.after(() => rmSync(root, { recursive: true, force: true }))
  for (const file of ['src/features/new-feature/reader.test.ts', 'src/shared/nested/parser.test.ts', 'scripts/tool.test.mjs', 'src/features/new-feature/reader.ts', 'src/fixtures/example.json', 'src/browser/example.spec.ts']) {
    mkdirSync(dirname(join(root, file)), { recursive: true })
    writeFileSync(join(root, file), '')
  }
  assert.deepEqual(discoverTests(root), ['scripts/tool.test.mjs', 'src/features/new-feature/reader.test.ts', 'src/shared/nested/parser.test.ts'])
})
