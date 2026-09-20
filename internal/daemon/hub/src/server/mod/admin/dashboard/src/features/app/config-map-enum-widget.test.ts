import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EditorState } from '@uiw/react-codemirror'
import { configMapEntries } from './config-map-enum.ts'
import { createConfigMapEnumExtension } from './config-map-enum-widget.ts'
import { createConfigJsonDocument, normalizeConfigJson } from './config-json-document.ts'
import { createConfigYamlDocument, normalizeConfigYaml } from './config-yaml-document.ts'
import { createConfigValueProtection } from './config-json-protection.ts'

const field = {
  name: 'statuses', type: 'map<demo.Region, demo.Status>', description: '',
  mapKeyEnumItems: [{ name: 'EAST', description: 'East' }, { name: 'WEST', description: 'West' }, { name: 'NORTH', description: 'North' }],
  mapValueEnumItems: [{ name: 'ACTIVE', description: 'Active' }, { name: 'LOCKED', description: 'Locked' }],
}

for (const yaml of [false, true]) {
  test(`${yaml ? 'YAML' : 'JSON5'} map dropdowns preserve structure and exclude duplicate keys`, () => {
    const value = '{"statuses":{"EAST":"ACTIVE","WEST":"LOCKED"}}'
    const doc = yaml ? createConfigYamlDocument(value, [field], true) : createConfigJsonDocument(value, [field])
    const protection = createConfigValueProtection(doc.ranges)
    const extensions = createConfigMapEnumExtension([field], protection.ranges, false, yaml, new Map(), new Set(['statuses']))
    let state = EditorState.create({ doc: doc.doc, extensions: [...protection.extensions, ...extensions] })
    const widgets = state.field(extensions[1])
    assert.equal(widgets.size, 4)
    const first = widgets.iter().value!.spec.widget
    assert.equal(first.dirty, true)
    assert.deepEqual(first.choices.map((choice: { label: string }) => choice.label), ['EAST', 'NORTH'])
    let entries = configMapEntries(state.doc.toString(), state.field(protection.ranges)[0], yaml)
    state = state.update({ changes: { ...entries[0].keyRange, insert: '"NORTH"' } }).state
    entries = configMapEntries(state.doc.toString(), state.field(protection.ranges)[0], yaml)
    assert.equal(entries[0].key, 'NORTH')
    state = state.update({ changes: { ...entries[0].valueRange, insert: '"LOCKED"' } }).state
    assert.deepEqual(JSON.parse((yaml ? normalizeConfigYaml : normalizeConfigJson)(state.doc.toString())), { statuses: { NORTH: 'LOCKED', WEST: 'LOCKED' } })
    assert.equal(state.field(extensions[1]).size, 4)
  })
}

test('string map keys remain editable and invalid enum values have read-only dropdowns', () => {
  const schema = { ...field, type: 'map<string, demo.Status>', mapKeyEnumItems: [] }
  const doc = createConfigJsonDocument('{"statuses":{"primary":123}}', [schema])
  const protection = createConfigValueProtection(doc.ranges)
  const extensions = createConfigMapEnumExtension([schema], protection.ranges, true, false, new Map([['statuses', 'Invalid enum']]), new Set())
  const state = EditorState.create({ doc: doc.doc, extensions: [...protection.extensions, ...extensions] })
  const widgets = state.field(extensions[1])
  assert.equal(widgets.size, 1)
  assert.equal(widgets.iter().value!.spec.widget.readOnly, true)
  assert.equal(widgets.iter().value!.spec.widget.error, 'Invalid enum')
})

test('YAML numeric map keys retain enum value dropdowns and exact edit ranges', () => {
  const schema = { ...field, type: 'map<int, demo.Status>', mapKeyEnumItems: [] }
  const text = '1: ACTIVE\n9007199254740993: LOCKED'
  const protection = createConfigValueProtection([{ name: 'statuses', from: 0, to: text.length }])
  const extensions = createConfigMapEnumExtension([schema], protection.ranges, false, true, new Map(), new Set())
  let state = EditorState.create({ doc: text, extensions: [...protection.extensions, ...extensions] })
  assert.equal(state.field(extensions[1]).size, 2)
  const entries = configMapEntries(text, state.field(protection.ranges)[0], true)
  assert.equal(entries[1].key, '9007199254740993')
  state = state.update({ changes: { ...entries[0].valueRange, insert: 'LOCKED' } }).state
  assert.deepEqual(JSON.parse(normalizeConfigYaml(state.doc.toString())), { '1': 'LOCKED', '9007199254740993': 'LOCKED' })
  assert.equal(state.field(extensions[1]).size, 2)
})
