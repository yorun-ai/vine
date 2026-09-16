import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { conflictsByRuleId } from './rule-conflicts.ts'

test('indexes a conflict by both rules it names', () => {
  const conflict = {
    ruleId: 2,
    rule: 'fixed',
    conflictRuleId: 1,
    conflictRule: 'web',
    entry: 'http:7088',
    match: 'http://*:7088/',
    publishedRuleId: 1,
    publishedRule: 'web',
    suppressedRuleId: 2,
    suppressedRule: 'fixed',
  }

  const indexed = conflictsByRuleId([conflict])

  assert.equal(indexed.get(1), conflict)
  assert.equal(indexed.get(2), conflict)
  assert.equal(indexed.size, 2)
})

test('reports no rule without a conflict', () => {
  assert.equal(conflictsByRuleId([]).size, 0)
})
