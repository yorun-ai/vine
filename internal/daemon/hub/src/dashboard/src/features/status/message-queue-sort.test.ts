import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import type { MessageQueueConsumer } from '../../skeled/admin/data.ts'
import {
  nextQueueSort,
  sortQueueConsumers,
  sortQueueSubjects,
} from './message-queue-sort.ts'

test('sorts message counts numerically without losing uint64 precision or changing the snapshot', () => {
  const rows = [
    { subject: 'task.b', messages: '9007199254740993' },
    { subject: 'task.c', messages: '2' },
    { subject: 'task.a', messages: '9007199254740992' },
    { subject: 'task.d', messages: '10' },
  ]
  const original = structuredClone(rows)
  assert.deepEqual(
    sortQueueSubjects(rows, { key: 'messages', direction: 'desc' }).map(
      (row) => row.subject,
    ),
    ['task.b', 'task.a', 'task.d', 'task.c'],
  )
  assert.deepEqual(
    sortQueueSubjects(rows, { key: 'messages', direction: 'asc' }).map(
      (row) => row.subject,
    ),
    ['task.c', 'task.d', 'task.a', 'task.b'],
  )
  assert.deepEqual(
    sortQueueSubjects(rows, { key: 'subject', direction: 'desc' }).map(
      (row) => row.subject,
    ),
    ['task.d', 'task.c', 'task.b', 'task.a'],
  )
  assert.deepEqual(rows, original)
})

const consumer = (
  name: string,
  subject: string,
  count: number,
): MessageQueueConsumer => ({
  name,
  filterSubjects: [subject],
  pending: String(count),
  ackPending: count,
  redelivered: count,
  waiting: count,
})

test('sorts consumers by their subjects independently of consumer names and uses stable ties', () => {
  const rows = [
    consumer('a', 'event.z', 2),
    consumer('c', 'event.a', 10),
    consumer('b', 'event.a', 10),
  ]
  assert.deepEqual(
    sortQueueConsumers(rows, { key: 'subject', direction: 'asc' }).map(
      (row) => row.name,
    ),
    ['b', 'c', 'a'],
  )
  assert.deepEqual(
    sortQueueConsumers(rows, { key: 'name', direction: 'asc' }).map(
      (row) => row.name,
    ),
    ['a', 'b', 'c'],
  )
  for (const key of [
    'pending',
    'ackPending',
    'redelivered',
    'waiting',
  ] as const) {
    assert.deepEqual(
      sortQueueConsumers(rows, { key, direction: 'desc' }).map(
        (row) => row.name,
      ),
      ['b', 'c', 'a'],
    )
    assert.deepEqual(
      sortQueueConsumers([...rows].reverse(), { key, direction: 'desc' }),
      sortQueueConsumers(rows, { key, direction: 'desc' }),
    )
  }
  assert.deepEqual(
    rows.map((row) => row.name),
    ['a', 'c', 'b'],
  )
})

test('sorts large pending counts exactly and matches all subject filters', () => {
  const rows = [
    {
      ...consumer('a', 'event.a', 0),
      pending: '18446744073709551614',
      filterSubjects: ['event.a', 'event.z'],
    },
    {
      ...consumer('b', 'event.a', 0),
      pending: '18446744073709551615',
      filterSubjects: ['event.a', 'event.b'],
    },
  ]
  for (const key of ['pending', 'subject'] as const) {
    const direction = key === 'pending' ? 'desc' : 'asc'
    assert.deepEqual(
      sortQueueConsumers(rows, { key, direction }).map((row) => row.name),
      ['b', 'a'],
    )
  }
})

test('column selection starts counts descending and subjects ascending, then toggles', () => {
  assert.deepEqual(
    nextQueueSort({ key: 'subject', direction: 'asc' }, 'messages'),
    { key: 'messages', direction: 'desc' },
  )
  assert.deepEqual(
    nextQueueSort({ key: 'messages', direction: 'desc' }, 'messages'),
    { key: 'messages', direction: 'asc' },
  )
  assert.deepEqual(
    nextQueueSort({ key: 'messages', direction: 'asc' }, 'subject'),
    { key: 'subject', direction: 'asc' },
  )
  assert.deepEqual(
    nextQueueSort({ key: 'subject', direction: 'asc' }, 'subject'),
    { key: 'subject', direction: 'desc' },
  )
})
