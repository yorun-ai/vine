import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { filterPortalInstances } from './instance-filter.ts'

const instances = [
  { instanceId: 'portal-a', version: '1.2.3' },
  { instanceId: 'Portal-B', version: '1.2.4' },
  { instanceId: 'edge-c', version: '' },
]

test('returns every instance for a blank query', () => {
  for (const query of ['', '   ']) {
    assert.deepEqual(filterPortalInstances(instances, query), instances, query)
  }
})

test('matches instance ids and versions case-insensitively', () => {
  assert.deepEqual(filterPortalInstances(instances, 'PORTAL-b'), [instances[1]])
  assert.deepEqual(filterPortalInstances(instances, '1.2.'), [instances[0], instances[1]])
  assert.deepEqual(filterPortalInstances(instances, 'edge'), [instances[2]])
})

test('trims the query, keeps the source order, and reports no match', () => {
  assert.deepEqual(filterPortalInstances(instances, '  1.2.4  '), [instances[1]])
  assert.deepEqual(filterPortalInstances(instances, 'portal'), [instances[0], instances[1]])
  assert.deepEqual(filterPortalInstances(instances, 'missing'), [])
})
