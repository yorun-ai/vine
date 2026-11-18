import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { configValueIssues } from './config-value-validation.ts'

const cases: Array<{ type: string; value: unknown; valid: boolean }> = JSON.parse(readFileSync(new URL('./testdata/config-scalar.json', import.meta.url), 'utf8'))
for (const item of cases) {
  test(`${item.type} format: ${JSON.stringify(item.value)}`, () => {
    const field = { name: 'value', type: item.type, description: '' }
    assert.equal(configValueIssues(item.value, field).length === 0, item.valid)
    assert.equal(configValueIssues([item.value], { ...field, type: `list<${item.type}>` }).length === 0, item.valid)
    assert.equal(configValueIssues({ key: item.value }, { ...field, type: `map<string, ${item.type}>` }).length === 0, item.valid)
  })
}

test('each nullable scalar type accepts null', () => {
  for (const type of new Set(cases.map((item) => item.type))) {
    assert.deepEqual(configValueIssues(null, { name: 'value', type: `${type}?`, description: '' }), [], type)
  }
})
