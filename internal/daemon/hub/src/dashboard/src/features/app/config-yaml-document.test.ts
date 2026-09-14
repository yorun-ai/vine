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
  assert.equal(dirty[0], doc.doc.indexOf('# @type list<booker.Category>'))
  assert.equal(dirty.length, 5)
})

test('YAML diagnostics reject duplicate keys, non-finite numbers, custom tags and invalid keys', () => {
  for (const text of ['a: 1\na: 2', 'a: .inf', 'a: .nan', 'a: !custom x', 'null: value', '? [a, b]: value', '&loop {self: *loop}', 'a: [unterminated']) {
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

test('unsafe YAML integers block saving and preserve backend text', () => {
  for (const number of ['9007199254740993', '-9007199254740993', '1e20']) {
    assert.throws(() => normalizeConfigYaml(`nested: [${number}]`), /safe range|Unsupported YAML number/)
    const raw = `{"value":${number}}`
    assert.equal(createConfigYamlDocument(raw, [], false).doc, raw)
  }
  assert.doesNotThrow(() => normalizeConfigYaml('max: 9007199254740991\nmin: -9007199254740991\nfraction: 1.5\ntext: "9007199254740993"'))
})


test('YAML normalizes scalar keys without losing numeric spelling or precision', () => {
  assert.deepEqual(parseConfigYaml('keys: {1: one, -2: two, 9007199254740993: large}'), {
    keys: { '1': 'one', '-2': 'two', '9007199254740993': 'large' },
  })
  assert.deepEqual(parseConfigYaml('"<<": literal'), { '<<': 'literal' })
})

test('YAML rejects merge keys and collisions after normalizing keys', () => {
  for (const text of ['keys: {1: one, "1": two}', '<<: {enabled: true}', 'nested: {<<: {enabled: true}}']) {
    assert.throws(() => parseConfigYaml(text))
    const errors = getConfigYamlErrors(text)
    assert.equal(errors.length, 1)
    assert.ok(errors[0].to > errors[0].from)
  }
})

test('YAML rejects anchors and aliases including unused anchors', () => {
  for (const text of ['a: &value 1', '&value [1, 2]', '&value {a: 1}', 'a: &value 1\nb: *value', '&loop {self: *loop}']) {
    assert.throws(() => parseConfigYaml(text), /anchors and aliases are not supported/)
    assert.match(getConfigYamlErrors(text)[0].message, /anchors and aliases are not supported/)
  }
})

test('YAML rejects ambiguous leading-zero integer values but preserves keys and strings', () => {
  for (const text of ['count: 012', 'count: -012', 'count: +012', 'count: 0_12', 'count: 019', 'values: [012]', 'count: 1_000', 'count: 0b10', 'count: 0x10', 'count: 0o12', 'count: 1e3', 'count: .5', '0x10: value']) {
    assert.throws(() => parseConfigYaml(text), /Unsupported YAML number/)
  }
  assert.deepEqual(parseConfigYaml('count: 10\nplain: 12\ntext: "012"\nkeys: {"012": value}'), {
    count: 10, plain: 12, text: '012', keys: { '012': 'value' },
  })
})

test('generated YAML expands scientific notation into ordinary decimals', () => {
  const value = { small: 1e-7, negative: -1e-10, smallest: Number.MIN_VALUE }
  const text = formatConfigYaml(JSON.stringify(value))
  assert.doesNotMatch(text, /[eE][+-]?\d/)
  assert.deepEqual(parseConfigYaml(text), value)
  const doc = createConfigYamlDocument(JSON.stringify(value), [], true)
  assert.deepEqual(parseConfigYaml(doc.doc), value)
})

test('YAML source comments do not enter configuration values', () => {
  const document = createConfigYamlDocument('{"enabled":true}', [{
    name: 'enabled', type: 'bool', description: 'Enabled',
    commentTags: ['type', 'desc', 'define', 'override', 'source'],
    sourceComment: '@define domain/user\n@variables ENABLED',
  }], true)
  assert.ok(document.doc.includes('# @define    domain/user'))
  assert.ok(document.doc.includes('# @variables ENABLED'))
  assert.deepEqual(parseConfigYaml(document.doc), { enabled: true })
  assert.equal(document.doc.slice(document.ranges[0].from, document.ranges[0].to).trim(), 'true')
})
