import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { isValidConfigSkelName } from './config-name.ts'

const cases: Array<{ name: string; valid: boolean }> = JSON.parse(
  readFileSync(new URL('./testdata/config-name.json', import.meta.url), 'utf8'),
)

test('config creation names match the Hub contract, including version suffixes', () => {
  for (const { name, valid } of cases) {
    assert.equal(isValidConfigSkelName(name), valid, name)
  }
})
