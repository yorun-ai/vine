import assert from 'node:assert/strict'
import { test } from 'node:test'
import { formatConfigDuration, splitConfigDuration } from './config-json-duration.ts'
import { getConfigJsonChoices } from './config-json-choice.ts'

test('custom duration values preserve units, fractions, zero and sign', () => {
  for (const [amount, unit] of [['250', 'ms'], ['1.5', 's'], ['0', 'h'], ['-2', 'm'], ['0.001', 'us']]) {
    assert.equal(formatConfigDuration(amount, unit), `${amount}${unit}`)
    assert.deepEqual(splitConfigDuration(`${amount}${unit}`), { amount, unit })
  }
  assert.deepEqual(splitConfigDuration('5µs'), { amount: '5', unit: 'us' })
})

test('empty or unsupported inputs cannot be applied', () => {
  for (const amount of ['', 'abc', 'NaN', 'Infinity', '1e3', '1s']) {
    assert.equal(formatConfigDuration(amount, 's'), null)
  }
  assert.equal(formatConfigDuration('1', 'days'), null)
  assert.deepEqual(splitConfigDuration('1h30m'), { amount: '', unit: 's' })
  assert.deepEqual(splitConfigDuration(null), { amount: '', unit: 's' })
})

test('duration has no presets, retaining only null for optional values', () => {
  assert.deepEqual(getConfigJsonChoices({ name: 'timeout', type: 'duration', description: '' }), [])
  assert.deepEqual(getConfigJsonChoices({ name: 'timeout', type: 'duration?', description: '' }), [
    { value: 'null', label: 'null' },
  ])
})
