import assert from 'node:assert/strict'
import { test } from 'node:test'
import { javascriptLanguage } from '@codemirror/lang-javascript'
import { EditorState } from '@uiw/react-codemirror'
import { createConfigJsonDocument, extractConfigJson, getConfigJsonErrors, getConfigJsonTypeLinks } from './config-json-document.ts'
import { createConfigValueProtection } from './config-json-protection.ts'

test('editor transactions lock keys, descriptions and structural punctuation', () => {
  const document = createConfigJsonDocument('{"port":7099,"host":"localhost"}', [
    { name: 'port', type: 'int', description: 'Port number' },
  ])
  const protection = createConfigValueProtection(document.ranges)
  const state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  for (const [from, to] of [
    [0, document.doc.length],
    [document.doc.indexOf('"port"'), document.doc.indexOf('"port"') + 6],
    [document.doc.indexOf('Port number'), document.doc.indexOf('Port number') + 11],
    [document.ranges[0].to, document.ranges[0].to + 1],
  ]) {
    assert.equal(state.update({ changes: { from, to, insert: 'changed' } }).docChanged, false)
  }
})

test('value ranges follow edits, including empty drafts and later fields', () => {
  const document = createConfigJsonDocument('{"port":7099,"host":"localhost"}', [])
  const protection = createConfigValueProtection(document.ranges)
  let state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  const edit = (index: number, insert: string) => {
    const range = state.field(protection.ranges)[index]
    state = state.update({ changes: { from: range.from, to: range.to, insert } }).state
  }
  edit(0, '')
  assert.throws(() => extractConfigJson(state.doc.toString(), state.field(protection.ranges)))
  edit(0, '80')
  edit(1, '"example.local"')
  assert.deepEqual(JSON.parse(extractConfigJson(state.doc.toString(), state.field(protection.ranges))), {
    port: 80,
    host: 'example.local',
  })
  assert.equal(state.doc.toString().includes('\n\n  "host"'), true)
})

test('multi-selection edits are rejected if any selection touches protected text', () => {
  const document = createConfigJsonDocument('{"port":7099}', [])
  const protection = createConfigValueProtection(document.ranges)
  const state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  const range = document.ranges[0]
  assert.equal(state.update({ changes: [
    { from: 0, to: 1, insert: '[' },
    { from: range.from, to: range.to, insert: '80' },
  ] }).docChanged, false)
})


test('editor grammar recognizes JSON5 comments, object keys and value syntax', () => {
  const language = javascriptLanguage.configure({ top: 'SingleExpression' })
  const tree = language.parser.parse("{ // description\n key: 'value', array: [0x50, +1, .5,], }")
  const names: Array<string> = []
  tree.iterate({ enter: (node) => {
    assert.equal(node.type.isError, false)
    names.push(node.name)
  } })
  assert.equal(names.includes('LineComment'), true)
  assert.equal(names.includes('PropertyDefinition'), true)
})

test('an unfinished string highlights only its own value, even before more fields', () => {
  const document = createConfigJsonDocument('{"timezone":"Asia/Shanghai","email":"hello@example.com"}', [])
  const protection = createConfigValueProtection(document.ranges)
  let state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  const range = state.field(protection.ranges)[0]
  state = state.update({ changes: { from: range.to - 1, to: range.to } }).state
  const errors = getConfigJsonErrors(state.doc.toString(), state.field(protection.ranges))
  assert.equal(errors.length, 1)
  assert.equal(errors[0].name, 'timezone')
  assert.equal(state.doc.sliceString(errors[0].from, errors[0].to), '"Asia/Shanghai')
  const end = state.field(protection.ranges)[0].to
  state = state.update({ changes: { from: end, insert: '"' } }).state
  assert.deepEqual(getConfigJsonErrors(state.doc.toString(), state.field(protection.ranges)), [])
})

test('all invalid values are located independently, including empty and non-finite drafts', () => {
  const doc = '{"a": , "b": NaN, "c": true}'
  const ranges = [
    { name: 'a', from: 6, to: 6 },
    { name: 'b', from: 13, to: 16 },
    { name: 'c', from: doc.indexOf('true'), to: doc.indexOf('true') + 4 },
  ]
  assert.deepEqual(getConfigJsonErrors(doc, ranges).map((error) => error.name), ['a', 'b'])
})

test('type links stay attached to their comments as earlier values grow', () => {
  const document = createConfigJsonDocument('{"name":"short","category":"A"}', [
    { name: 'name', type: 'string', description: '' },
    { name: 'category', type: 'booker.Category', description: '' },
  ])
  const protection = createConfigValueProtection(document.ranges)
  let state = EditorState.create({ doc: document.doc, extensions: protection.extensions })
  const range = state.field(protection.ranges)[0]
  state = state.update({ changes: { from: range.from, to: range.to, insert: '"a much longer value"' } }).state
  const definitions = new Map([['booker.Category', { skelName: 'booker.Category' }]])
  const [link] = getConfigJsonTypeLinks(document, state.field(protection.ranges), definitions)
  assert.equal(state.doc.sliceString(link.from, link.to), 'booker.Category')
  assert.equal(state.update({ changes: { from: link.from, to: link.to, insert: 'changed' } }).docChanged, false)
})
