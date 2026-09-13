import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EditorState } from '@uiw/react-codemirror'
import { createConfigJsonDocument, extractConfigJson } from './config-json-document.ts'
import { createConfigValueProtection } from './config-json-protection.ts'
import { createConfigChoiceExtension } from './config-json-choice-widget.ts'

test('multiline enum lists use direct decorations and preserve JSON arrays', () => {
  const fields = [{ name: 'categories', type: 'list<booker.Category>', description: '', enumItems: [
    { name: 'A', description: 'First' }, { name: 'B', description: 'Second' },
  ] }]
  const document = createConfigJsonDocument('{"categories":["A","B"]}', fields)
  const protection = createConfigValueProtection(document.ranges)
  const choices = createConfigChoiceExtension(fields, false, protection.ranges)
  let state = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, choices] })
  assert.equal(state.field(choices).size, 1)
  const range = state.field(protection.ranges)[0]
  assert.equal(state.doc.sliceString(range.from, range.to).includes('\n'), true)
  state = state.update({ changes: { from: range.from, to: range.to, insert: '["B"]' } }).state
  assert.equal(state.field(choices).size, 1)
  assert.deepEqual(JSON.parse(extractConfigJson(state.doc.toString(), state.field(protection.ranges))), { categories: ['B'] })
})

test('mismatched choices expose their field error and clear it after correction', () => {
  const fields = [{ name: 'enabled', type: 'bool', description: '' }]
  const document = createConfigJsonDocument('{"enabled":"yes"}', fields)
  const protection = createConfigValueProtection(document.ranges)
  const message = 'enabled: expected bool, got string'
  const choices = createConfigChoiceExtension(fields, false, protection.ranges, new Map([['enabled', message]]))
  const state = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, choices] })
  assert.equal(state.field(choices).iter().value!.spec.widget.error, message)
  const corrected = createConfigChoiceExtension(fields, false, protection.ranges)
  const next = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, corrected] })
  assert.equal(next.field(corrected).iter().value!.spec.widget.error, '')
})

test('duration fields receive a duration editor while retaining their existing value', () => {
  const fields = [{ name: 'timeout', type: 'duration', description: '' }]
  const document = createConfigJsonDocument('{"timeout":"1h30m"}', fields)
  const protection = createConfigValueProtection(document.ranges)
  const choices = createConfigChoiceExtension(fields, false, protection.ranges)
  const state = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, choices] })
  const widget = state.field(choices).iter().value!.spec.widget
  assert.equal(widget.duration, true)
  assert.equal(widget.value, '"1h30m"')
  assert.equal(state.doc.toString(), document.doc)
})

test('changed dropdowns retain their dirty state alongside error priority', () => {
  const fields = [{ name: 'enabled', type: 'bool', description: '' }]
  const document = createConfigJsonDocument('{"enabled":true}', fields)
  const protection = createConfigValueProtection(document.ranges)
  const choices = createConfigChoiceExtension(fields, false, protection.ranges, new Map(), undefined, new Set(['enabled']))
  const state = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, choices] })
  assert.equal(state.field(choices).iter().value!.spec.widget.dirty, true)
})

test('all date-time types have specialized editors, including nullable fields', () => {
  for (const type of ['localdate', 'localtime', 'localdatetime', 'timestamp', 'timestamp?']) {
    const fields = [{ name: 'schedule', type, description: '' }]
    const document = createConfigJsonDocument('{"schedule":null}', fields)
    const protection = createConfigValueProtection(document.ranges)
    const choices = createConfigChoiceExtension(fields, false, protection.ranges)
    const state = EditorState.create({ doc: document.doc, extensions: [...protection.extensions, choices] })
    assert.equal(state.field(choices).iter().value!.spec.widget.dateTimeType, type)
  }
})
