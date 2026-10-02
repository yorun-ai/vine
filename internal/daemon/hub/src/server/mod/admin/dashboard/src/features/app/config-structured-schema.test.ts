import assert from 'node:assert/strict'
import { test } from 'node:test'
import { configValueIssues } from './config-value-validation.ts'
import { configSchemaHints, configSchemaCompletions } from './config-schema-hints.ts'
import { defaultStructuredConfigValue, configTypeText, validConfigBinary } from './config-structured-schema.ts'
import type { ConfigSchemaType, ConfigSchemaData } from './config-structured-schema.ts'
import { parseConfigYaml } from './config-yaml-document.ts'

const type = (kind: string, name = '', changes: Partial<ConfigSchemaType> = {}): ConfigSchemaType => ({
  kind, name, nullable: false, typeArguments: [], element: null, key: null, value: null, enumItems: [], ...changes,
})
const binary = type('scalar', 'binary')
const leaf = type('data', 'demo.Leaf')
const box = type('data', 'demo.Box', { typeArguments: [leaf] })
const data: ConfigSchemaData[] = [
  { skelName: 'demo.Box', sensitive: false, typeParameters: ['T'], fields: [{ name: 'value', type: 'T', description: 'Payload', valueType: type('typeParameter', 'T') }] },
  { skelName: 'demo.Leaf', sensitive: false, typeParameters: [], fields: [
    { name: 'bytes', type: 'binary', description: 'Certificate', valueType: binary },
    { name: 'mode', type: 'demo.Mode', description: 'Operation mode', valueType: type('enum', 'demo.Mode', { enumItems: [{ name: 'ACTIVE', description: 'Active' }] }) },
    { name: 'next', type: 'demo.Leaf?', description: '', valueType: { ...leaf, nullable: true } },
  ] },
]
const field = { name: 'settings', type: 'demo.Box<demo.Leaf>', description: '', valueType: box, dataTypes: data }
const valid = { value: { bytes: 'aG\r\nVsbG8=', mode: 'ACTIVE', next: null } }

test('generic data defaults and validation resolve nested declarations', () => {
  assert.equal(configTypeText(box), 'demo.Box<demo.Leaf>')
  assert.deepEqual(defaultStructuredConfigValue(box, data), { value: { bytes: '', mode: 'ACTIVE', next: null } })
  assert.deepEqual(configValueIssues(valid, field), [])
  assert.deepEqual(configValueIssues({ value: { bytes: '%%%', mode: 'BAD', extra: true } }, field).map((issue) => issue.path), [
    'settings.value.bytes', 'settings.value.mode', 'settings.value.next', 'settings.value["extra"]',
  ])
  assert.equal(configValueIssues(valid, field, []).length, 1)
  assert.equal(configValueIssues(null, field).length, 1)
})

test('data through maps and lists and inherited sensitive metadata', () => {
  const collection = type('map', '', { key: type('scalar', 'string'), value: type('list', '', { element: box }) })
  assert.deepEqual(configValueIssues({ key: [valid] }, { ...field, valueType: collection }), [])
  const sensitiveData = data.map((item) => ({ ...item, sensitive: true }))
  const issues = configValueIssues({ value: { bytes: 'PRIVATE', mode: 'PRIVATE', next: null, PRIVATE: 'PRIVATE' } }, field, sensitiveData)
  assert.equal(JSON.stringify(issues).includes('PRIVATE'), false)
  const sensitiveMap = { ...field, sensitive: true, valueType: type('map', '', { key: type('scalar', 'int'), value: binary }) }
  assert.equal(JSON.stringify(configValueIssues({ PRIVATE: 'PRIVATE' }, sensitiveMap)).includes('PRIVATE'), false)
  const inheritedSensitiveMap = { ...field, valueType: collection, dataTypes: sensitiveData }
  const inheritedIssues = configValueIssues({ PRIVATE: [{ value: { bytes: 'PRIVATE', mode: 'PRIVATE', next: null } }] }, inheritedSensitiveMap)
  assert.ok(inheritedIssues.length > 0)
  assert.equal(JSON.stringify(inheritedIssues).includes('PRIVATE'), false)
})

test('binary accepts CR/LF and padding, rejecting other whitespace and alphabets', () => {
  for (const item of ['', 'aGVsbG8=', 'aG\nVsbG8=', 'aG\r\nVsbG8=']) assert.equal(validConfigBinary(item), true)
  for (const item of ['aG VsbG8=', 'aG\tVsbG8=', 'aGVsbG8', '====', 'aG-sbG8=', 12, null]) assert.equal(validConfigBinary(item), false)
  const yaml = parseConfigYaml('settings:\n  value:\n    bytes: |\n      aG\n      VsbG8=\n    mode: ACTIVE\n    next: null\n') as { settings: unknown }
  assert.deepEqual(configValueIssues(yaml.settings, field), [])
})

test('nested JSON5 and YAML field hints show substituted types and enum descriptions', () => {
  const json = '{settings: {value: {bytes: "", mode: "ACTIVE", next: null}}}'
  const hints = configSchemaHints(json, [field])
  assert.ok(hints.some((hint) => hint.text.includes('binary\nCertificate')))
  assert.ok(hints.some((hint) => hint.text.includes('ACTIVE: Active')))
  assert.equal(hints.find((hint) => json.slice(hint.from, hint.to) === 'value')?.text, 'demo.Leaf\nPayload')
  const yaml = 'settings:\n  value:\n    bytes: ""\n    mode: ACTIVE\n    next: null\n'
  assert.ok(configSchemaHints(yaml, [field], true).some((hint) => hint.text.includes('binary\nCertificate')))
})


test('nested property and enum completion uses resolved generic members', () => {
  const json = '{settings: {value: { }}}'
  assert.deepEqual(configSchemaCompletions(json, json.indexOf(' }'), [field]).map((item) => item.label), ['bytes', 'mode', 'next'])
  const enumDoc = '{settings: {value: {mode: "AC"}}}'
  assert.deepEqual(configSchemaCompletions(enumDoc, enumDoc.indexOf('AC') + 2, [field]).map((item) => item.label), ['ACTIVE'])
  const yaml = 'settings:\n  value:\n    '
  assert.deepEqual(configSchemaCompletions(yaml, yaml.length, [field], true).map((item) => item.label), ['bytes', 'mode', 'next'])
  const yamlEnum = 'settings:\n  value:\n    mode: AC'
  assert.deepEqual(configSchemaCompletions(yamlEnum, yamlEnum.length, [field], true).map((item) => item.label), ['ACTIVE'])
  const listField = { ...field, valueType: type('list', '', { element: leaf }) }
  const yamlList = 'settings:\n  - bytes: ""\n    '
  assert.deepEqual(configSchemaCompletions(yamlList, yamlList.length, [listField], true).map((item) => item.label), ['bytes', 'mode', 'next'])
  for (const name of ['mode', '"mode"', "'mode'"]) {
    const draft = yamlList + name + ': AC'
    assert.deepEqual(configSchemaCompletions(draft, draft.length, [listField], true).map((item) => item.label), ['ACTIVE'])
  }
})

test('nullable parameter references combine with nullable and required arguments', () => {
  const optionalData: ConfigSchemaData[] = [{ skelName: 'demo.Optional', sensitive: false, typeParameters: ['TValue'], fields: [
    { name: 'required', type: 'TValue', description: '', valueType: type('typeParameter', 'TValue') },
    { name: 'optional', type: 'TValue?', description: '', valueType: type('typeParameter', 'TValue', { nullable: true }) },
  ] }]
  for (const nullable of [false, true]) {
    const instance = type('data', 'demo.Optional', { typeArguments: [{ ...binary, nullable }] })
    const definition = { name: 'settings', type: configTypeText(instance), description: '', valueType: instance, dataTypes: optionalData }
    assert.deepEqual(configValueIssues({ required: '', optional: null }, definition), [])
    assert.deepEqual(configValueIssues({ required: '', optional: '' }, definition), [])
    assert.equal(configValueIssues({ required: null, optional: null }, definition).length, nullable ? 0 : 1)
    assert.equal(configValueIssues({ required: '', optional: 'invalid base64' }, definition).length, 1)
    assert.deepEqual(defaultStructuredConfigValue(instance, optionalData), { required: nullable ? null : '', optional: null })
  }
})
