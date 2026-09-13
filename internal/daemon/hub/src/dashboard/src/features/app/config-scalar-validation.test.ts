import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { configScalarFormatMatches } from './config-scalar-validation.ts'
import { configValueIssues } from './config-value-validation.ts'

const cases: Array<{ type: string; value: unknown; valid: boolean }> = JSON.parse(readFileSync(new URL('./testdata/config-scalar.json', import.meta.url), 'utf8'))
for (const item of cases) {
  test(`${item.type} format: ${JSON.stringify(item.value)}`, () => {
    assert.equal(configScalarFormatMatches(item.value, item.type), item.valid)
    const field = { name: 'value', type: item.type, description: '' }
    assert.equal(configValueIssues(item.value, field).length === 0, item.valid)
    assert.equal(configValueIssues([item.value], { ...field, type: `list<${item.type}>` }).length === 0, item.valid)
    assert.equal(configValueIssues({ key: item.value }, { ...field, type: `map<string, ${item.type}>` }).length === 0, item.valid)
    assert.deepEqual(configValueIssues(null, { ...field, type: `${item.type}?` }), [])
  })
}
