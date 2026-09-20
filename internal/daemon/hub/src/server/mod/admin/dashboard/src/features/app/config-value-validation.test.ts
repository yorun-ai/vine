import assert from 'node:assert/strict'
import { test } from 'node:test'
import { configValueIssues } from './config-value-validation.ts'

const field = (type: string) => ({ name: 'settings', type, description: '' })

test('integer and collection diagnostics identify the offending entry', () => {
  assert.equal(configValueIssues(1.2, field('int')).length, 1)
  assert.deepEqual(configValueIssues(1.2, field('float')), [])
  assert.deepEqual(configValueIssues([1, 'two', 3.5], field('list<int>')).map((issue) => issue.path), ['settings[1]', 'settings[2]'])
  assert.deepEqual(configValueIssues({ bad: 'yes', '1': true }, field('map<int, bool>')).map((issue) => [issue.path, issue.part]), [['settings["bad"]', 'key'], ['settings["bad"]', 'value']])
  assert.deepEqual(configValueIssues({ '9007199254740993': true, '-9223372036854775808': false }, field('map<int, bool>')), [])
  for (const key of ['1.5', '01', '9223372036854775808', '-9223372036854775809']) {
    assert.equal(configValueIssues({ [key]: true }, field('map<int, bool>'))[0].part, 'key')
  }
})

test('nullable collections and enum elements retain their declared semantics', () => {
  const enumItems = [{ name: 'ACTIVE', description: '' }]
  assert.deepEqual(configValueIssues(null, field('list<int>?')), [])
  assert.equal(configValueIssues(null, field('list<int>')).length, 1)
  assert.deepEqual(configValueIssues([1, null], field('list<int?>')), [])
  const list = { ...field('list<demo.Status>'), enumItems }
  assert.deepEqual(configValueIssues(['ACTIVE'], list), [])
  assert.equal(configValueIssues(['BAD'], list)[0].path, 'settings[0]')
  const map = { ...field('map<demo.Status, demo.Status?>'), mapKeyEnumItems: enumItems, mapValueEnumItems: enumItems }
  assert.deepEqual(configValueIssues({ ACTIVE: null }, map), [])
  assert.equal(configValueIssues({ BAD: 'BAD' }, map).length, 2)
  assert.deepEqual(configValueIssues({ a: null }, field('map<string, string?>')), [])
})
