import assert from 'node:assert/strict'
import { test } from 'node:test'
import { configSourceComment } from './config-source-comments.ts'

const sources = [
  { path: '/value/accessTokenTTL', source: 'app/default', define: 'domain/user', override: 'app/default', variables: ['ACCESS_TTL'] },
  { path: '/value/refreshTokenTTL', source: 'domain/user', define: 'domain/user', override: '', variables: [] },
  { path: '/name', source: 'profile/dev', define: 'domain/user', override: 'profile/dev', variables: [] },
]

test('comments use each config value key, never the entity name or another value key', () => {
  assert.equal(configSourceComment('accessTokenTTL', sources), '@source app/default\n@define domain/user\n@override app/default\n@variables ACCESS_TTL')
  assert.equal(configSourceComment('refreshTokenTTL', sources), '@source domain/user\n@define domain/user\n@override ')
  assert.equal(configSourceComment('missing', sources), '@override ')
})

test('nested origins are not displayed and pointer segments are escaped', () => {
  const fields = [
    { path: '/value/options/keep', source: 'domain/user', define: 'domain/user', override: '', variables: [] },
    { path: '/value/options/change', source: 'profile/dev', define: 'domain/user', override: 'profile/dev', variables: [] },
    { path: '/value/a~1b~0c', source: 'app/yaoming', define: 'app/yaoming', override: '', variables: [] },
  ]
  assert.equal(configSourceComment('options', fields), '@override ')
  assert.equal(configSourceComment('a/b~c', fields), '@source app/yaoming\n@define app/yaoming\n@override ')
})

test('a whole-value variable source is inherited only without a closer field origin', () => {
  const fields = [{ path: '/value', source: 'profile/dev', define: 'profile/dev', override: '', variables: ['CONFIG'] }, ...sources]
  assert.equal(configSourceComment('refreshTokenTTL', fields), '@source domain/user\n@define domain/user\n@override ')
  assert.equal(configSourceComment('other', fields), '@source profile/dev\n@define profile/dev\n@override \n@variables CONFIG')
})

test('list and object sources display once without descendant paths', () => {
 for (const name of ['categories', 'options']) {
  assert.equal(configSourceComment(name, [{path: `/value/${name}`, source: 'app/default', define: 'domain/booker', override: 'app/default', variables: ['A', 'B']}]), '@source app/default\n@define domain/booker\n@override app/default\n@variables A, B')
 }
})

test('source is supplied by the backend without frontend fallback', () => {
  assert.equal(configSourceComment('enabled', [{path: '/value/enabled', source: '', define: 'domain/demo', override: 'app/default', variables: []}]), '@define domain/demo\n@override app/default')
})

test('template and resolved bindings preserve types and distinct defaults', () => {
 const comment = configSourceComment('options', [{
  path: '/value/options', source: 'app/default', define: 'app/default', override: '', variables: ['port'],
  template: '{"first":"${port:80}","second":"${port:443}"}',
  bindings: [
   {path: '/first', variable: 'port', reference: '${port:80}', value: '80', defaultUsed: true},
   {path: '/second', variable: 'port', reference: '${port:443}', value: '443', defaultUsed: true},
  ],
 }])
 assert.ok(comment?.includes('@template {"first":"${port:80}","second":"${port:443}"}'))
 assert.ok(comment?.includes('@variables /first: ${port:80} = 80 (default)'))
 assert.ok(comment?.includes('@variables /second: ${port:443} = 443 (default)'))
})
