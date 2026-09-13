import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EditorState } from '@uiw/react-codemirror'
import { createConfigYamlDocument, parseConfigYaml, normalizeConfigYaml, getConfigYamlErrors, formatConfigYaml } from './config-yaml-document.ts'
import { getConfigJsonDirtyLines, getConfigJsonTypeLinks } from './config-json-document.ts'
import { createConfigValueProtection } from './config-json-protection.ts'
import { createConfigChoiceExtension } from './config-json-choice-widget.ts'

const fields = [
  { name: 'enabled', type: 'bool', description: 'Enabled' },
  { name: 'categories', type: 'list<booker.Category>', description: 'Categories', enumItems: [{ name: 'A', description: '' }, { name: 'B', description: '' }] },
  { name: 'timeout', type: 'duration', description: 'Timeout' },
  { name: 'date', type: 'localdate', description: 'Date' },
]
const value = { enabled: true, categories: ['A', 'B'], timeout: '1h30m', date: '2026-09-13' }

test('YAML round-trips scalars, nested values, dates, multiline strings and quoted keys', () => {
  const object = { ...value, 'key: with punctuation': 'true', text: 'first\nsecond\n', object: { list: [null, 1, false] }, empty: {} }
  const doc = createConfigYamlDocument(JSON.stringify(object), fields, true)
  assert.deepEqual(parseConfigYaml(doc.doc), object)
  for (const range of doc.ranges) {
    assert.deepEqual(parseConfigYaml(doc.doc.slice(range.from, range.to)), object[range.name as keyof typeof object])
  }
  assert.deepEqual(JSON.parse(normalizeConfigYaml(formatConfigYaml(JSON.stringify(object)))), object)
})

test('YAML protects keys and supports scalar and block-list widgets', () => {
  const doc = createConfigYamlDocument(JSON.stringify(value), fields, true)
  const protection = createConfigValueProtection(doc.ranges)
  const widgets = createConfigChoiceExtension(fields, false, protection.ranges, new Map(), undefined, new Set(), true)
  let state = EditorState.create({ doc: doc.doc, extensions: [...protection.extensions, widgets] })
  assert.equal(state.field(widgets).size, 4)
  const key = doc.doc.indexOf('enabled:')
  assert.equal(state.update({ changes: { from: key, to: key + 7, insert: 'other' } }).state.doc.toString(), doc.doc)
  const range = doc.ranges[1]
  state = state.update({ changes: { from: range.from, to: range.to, insert: ' ["B"]' } }).state
  assert.deepEqual(parseConfigYaml(state.doc.toString()), { ...value, categories: ['B'] })
  assert.equal(state.field(widgets).size, 4)
})

test('YAML comments and type links participate in dirty blocks', () => {
  const doc = createConfigYamlDocument(JSON.stringify(value), fields, true)
  const links = getConfigJsonTypeLinks(doc, doc.ranges, new Map([['booker.Category', { skelName: 'booker.Category' }]]))
  assert.equal(doc.doc.slice(links[0].from, links[0].to), 'booker.Category')
  const dirty = getConfigJsonDirtyLines(doc.doc, doc.ranges, new Set(['categories']), new Map())
  assert.equal(dirty[0], doc.doc.indexOf('# list<booker.Category>'))
  assert.equal(dirty.length, 5)
})

test('YAML diagnostics reject duplicate keys, non-finite numbers, custom tags and non-string keys', () => {
  for (const text of ['a: 1\na: 2', 'a: .inf', 'a: .nan', 'a: !custom x', '1: value', 'a: [unterminated']) {
    assert.throws(() => normalizeConfigYaml(text))
    assert.ok(getConfigYamlErrors(text).length > 0)
  }
  const text = 'good: true\nbad: [unterminated'
  assert.ok(getConfigYamlErrors(text)[0].from >= text.indexOf('bad:'))
})

test('YAML reports fields inserted through a value range and preserves invalid replacement text', () => {
  const doc = createConfigYamlDocument('{"enabled":true}', fields, true)
  assert.match(getConfigYamlErrors('enabled: true\nextra: 1', doc.ranges)[0].message, /read-only/)
  assert.equal(createConfigYamlDocument('a: [broken', [], false).doc, 'a: [broken')
  assert.deepEqual(parseConfigYaml('message: yes\ndate: 2026-09-13'), { message: 'yes', date: '2026-09-13' })
})


test('empty YAML config objects stay objects and non-finite values have exact positions', () => {
  const document = createConfigYamlDocument('{}', [], true)
  assert.deepEqual(parseConfigYaml(document.doc), {})
  const text = 'enabled: true\nlimit: .inf'
  assert.equal(getConfigYamlErrors(text)[0].from, text.indexOf('.inf'))
})
