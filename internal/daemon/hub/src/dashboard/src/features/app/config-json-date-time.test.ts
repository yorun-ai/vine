import assert from 'node:assert/strict'
import { test } from 'node:test'
import { formatConfigDateTime, isConfigDateTime, splitConfigDateTime } from './config-json-date-time.ts'

test('date-time fields preserve fractions and timestamp offsets through edits', () => {
  const samples = [
    ['localdate', '2028-02-29'],
    ['localtime', '09:30:15.123456789'],
    ['localdatetime', '2026-10-01T18:30:00.250'],
    ['timestamp', '2026-09-13T10:15:30.123456789+08:00'],
    ['timestamp?', '2026-09-13T02:15:30Z'],
  ]
  for (const [type, value] of samples) {
    assert.equal(isConfigDateTime(type), true)
    assert.equal(formatConfigDateTime(type, splitConfigDateTime(type, value)), value)
  }
  const parts = splitConfigDateTime('timestamp', samples[3][1])
  parts.date = '2026-10-01'
  assert.equal(formatConfigDateTime('timestamp', parts), '2026-10-01T10:15:30.123456789+08:00')
})

test('invalid and incomplete dates and times are not applied', () => {
  for (const [type, value] of [
    ['localdate', '2026-02-29'], ['localdate', '2026-13-01'],
    ['localtime', '25:00:00'], ['localdatetime', '2026-10-01T12:60:00'],
    ['timestamp', '2026-10-01T12:00:00+25:00'],
  ]) {
    assert.equal(formatConfigDateTime(type, splitConfigDateTime(type, value)), null)
  }
  assert.equal(formatConfigDateTime('timestamp', splitConfigDateTime('timestamp', null)), null)
  assert.equal(isConfigDateTime('list<timestamp>'), false)
})
