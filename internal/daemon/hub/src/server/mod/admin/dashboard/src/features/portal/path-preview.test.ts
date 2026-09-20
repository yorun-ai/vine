import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { previewRulePath } from './path-preview.ts'

test('preserves queries, trailing slashes, and escaped suffixes like Portal', () => {
  const cases = [
    ['/api/users?x=1&next=%2Fhome', '/api', '/internal', '/internal/users?x=1&next=%2Fhome'],
    ['/api', '/api', '/internal/', '/internal'],
    ['/api/', '/api', '/internal', '/internal/'],
    ['/api', '/api', '', '/'],
    ['/api/users', '/api', '/', '/users'],
    ['/users', '/', '/internal', '/internal/users'],
    ['/users?x=1', '', '', '/users?x=1'],
    ['/api/a%2Fb/%25', '/api', '/internal', '/internal/a%2Fb/%25'],
    ['/%61pi/users', '/api', '/internal', '/internal/users'],
    ['/api%2Fusers', '/api', '/internal', '/internal%2Fusers'],
    ['/%E4%B8%AD%E6%96%87/users', '/中文', '/internal', '/internal/users'],
    ['/中文/users', '/中文', '/internal', '/internal/users'],
  ]
  for (const [request, match, route, path] of cases) {
    assert.deepEqual(previewRulePath(request, match, route), { kind: 'matched', path }, request)
  }
})

test('requires a path-segment match', () => {
  for (const path of ['/api2/users', '/other', '/API/users']) {
    assert.deepEqual(previewRulePath(path, '/api', '/internal'), { kind: 'noMatch' })
  }
})

test('reports invalid inputs without throwing', () => {
  assert.deepEqual(previewRulePath('', '/api', ''), { kind: 'empty' })
  for (const request of ['https://example.com/api', 'api', '/api#fragment', '/api/%zz', '/api/a b']) {
    assert.deepEqual(previewRulePath(request, '/api', ''), { kind: 'invalidRequest' }, request)
  }
  for (const route of ['internal', '//host', '/a/../b', '/%2e', '/a?b', '/a#b', '/a%5Cb', '/%zz']) {
    assert.deepEqual(previewRulePath('/api/users', '/api', route), { kind: 'invalidRoute' }, route)
  }
})
