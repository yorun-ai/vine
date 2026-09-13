import assert from 'node:assert/strict'
import { test } from 'node:test'
import { configMapEnumIssues } from './config-map-enum.ts'

const field = {
  name: 'statuses', type: 'map<demo.Region, demo.Status>', description: '',
  mapKeyEnumItems: [{ name: 'EAST', description: 'East' }, { name: 'WEST', description: 'West' }, { name: 'NORTH', description: 'North' }],
  mapValueEnumItems: [{ name: 'ACTIVE', description: 'Active' }, { name: 'LOCKED', description: 'Locked' }],
}

test('map enum diagnostics identify key and value errors and allow nullable values', () => {
  assert.deepEqual(configMapEnumIssues({ EAST: 'ACTIVE' }, field), [])
  const issues = configMapEnumIssues({ INVALID: 'BAD', EAST: 3 }, field)
  assert.deepEqual(issues.map((issue) => [issue.key, issue.part]), [['INVALID', 'key'], ['INVALID', 'value'], ['EAST', 'value']])
  assert.equal(configMapEnumIssues({ EAST: null }, field).length, 1)
  assert.deepEqual(configMapEnumIssues({ EAST: null }, { ...field, type: 'map<demo.Region, demo.Status?>' }), [])
})

