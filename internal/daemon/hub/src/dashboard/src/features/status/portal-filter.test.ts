import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { filterPortalInstances } from './portal-filter.ts'

const instances = [
  {
    instanceId: 'portal-a',
    version: '1.2.3',
    startedAt: '2026-09-15T06:30:00Z',
    inproc: false,
  },
  {
    instanceId: 'Portal-B',
    version: '1.2.4',
    startedAt: '2026-09-15T07:00:00Z',
    inproc: true,
  },
  {
    instanceId: 'edge-c',
    version: '',
    startedAt: '',
    inproc: false,
  },
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
