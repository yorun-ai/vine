import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EditorState } from '@uiw/react-codemirror'
import { getConfigJsonChoices, toggleConfigJsonChoice, isConfigJsonMultiChoice } from './config-json-choice.ts'
import { createConfigJsonDocument, extractConfigJson } from './config-json-document.ts'
import { createConfigValueProtection } from './config-json-protection.ts'

test('boolean options preserve JSON boolean types and optional null', () => {
  const field = { name: 'enabled', type: 'bool', description: '' }
  assert.deepEqual(getConfigJsonChoices(field).map((item) => JSON.parse(item.value)), [false, true])
  assert.deepEqual(getConfigJsonChoices({ ...field, type: 'bool?' }).map((item) => JSON.parse(item.value)), [false, true, null])
})

test('enum options preserve names and include descriptions', () => {
  const choices = getConfigJsonChoices({
    name: 'category', type: 'booker.Category?', description: '',
    enumItems: [{ name: 'LITERATURE', description: '文学' }, { name: 'SCIENCE', description: '' }],
  })
  assert.deepEqual(choices, [
    { value: '"LITERATURE"', label: 'LITERATURE', description: '文学' },
    { value: '"SCIENCE"', label: 'SCIENCE', description: '' },
    { value: 'null', label: 'null' },
  ])
})

test('collections and non-enum fields keep text editing', () => {
  for (const type of ['map<string, booker.Category>', 'list<list<booker.Category>>']) {
    assert.deepEqual(getConfigJsonChoices({
      name: 'items', type, description: '', enumItems: [{ name: 'A', description: '' }],
    }), [])
  }
  assert.deepEqual(getConfigJsonChoices({ name: 'title', type: 'string', description: '' }), [])
})

test('successive choices update current value ranges and save standard JSON', () => {
  const fields = [
    { name: 'enabled', type: 'bool', description: '' },
    { name: 'category', type: 'booker.Category', description: '', enumItems: [{ name: 'LITERATURE', description: '' }] },
  ]
  const document = createConfigJsonDocument('{"enabled":false,"category":"A"}', fields)
  const protection = createConfigValueProtection(document.ranges)
  let state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  for (const field of fields) {
    const choice = getConfigJsonChoices(field).at(-1)!
    const range = state.field(protection.ranges).find((item) => item.name === field.name)!
    state = state.update({ changes: { from: range.from, to: range.to, insert: choice.value } }).state
  }
  assert.deepEqual(JSON.parse(extractConfigJson(state.doc.toString(), state.field(protection.ranges))), {
    enabled: true,
    category: 'LITERATURE',
  })
})

test('enum lists expose multi-select choices and preserve enum descriptions separately', () => {
  const field = { name: 'categories', type: 'list<booker.Category>?', description: '', enumItems: [
    { name: 'SCIENCE', description: '科学' },
  ] }
  assert.equal(isConfigJsonMultiChoice(field), true)
  assert.deepEqual(getConfigJsonChoices(field), [
    { value: '"SCIENCE"', label: 'SCIENCE', description: '科学' },
    { value: 'null', label: 'null' },
  ])
  assert.equal(isConfigJsonMultiChoice({ ...field, type: 'booker.Category' }), false)
})

test('multi-select toggles values without duplicates and supports empty and null', () => {
  assert.deepEqual(toggleConfigJsonChoice(['A'], '"B"'), ['A', 'B'])
  assert.deepEqual(toggleConfigJsonChoice(['A', 'B'], '"A"'), ['B'])
  assert.deepEqual(toggleConfigJsonChoice(['A'], '"A"'), [])
  assert.deepEqual(toggleConfigJsonChoice(null, '"A"'), ['A'])
  assert.equal(toggleConfigJsonChoice(['A'], 'null'), null)
})
