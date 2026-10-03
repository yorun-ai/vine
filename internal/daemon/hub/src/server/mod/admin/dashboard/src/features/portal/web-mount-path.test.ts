import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { effectiveWebMountPrefixes, lockedWebMountPath } from './web-mount-path.ts'

const sites = [
  { name: 'demo', type: 'WEBGW', webMountPath: '/demo/' },
  { name: 'plain', type: 'WEBGW', webMountPath: '' },
  { name: 'api', type: 'RPCGW', webMountPath: '/demo/' },
]

test('locks prefixes of rules that target a site with a mount path', () => {
  assert.equal(lockedWebMountPath('SITE', 'demo', sites), '/demo/')
  assert.equal(lockedWebMountPath('SITE', 'plain', sites), null)
  assert.equal(lockedWebMountPath('SITE', 'api', sites), null)
  assert.equal(lockedWebMountPath('SITE', 'unknown', sites), null)
  assert.equal(lockedWebMountPath('SITE', '', sites), null)
  assert.equal(lockedWebMountPath('PERMANENT_REDIRECT', 'demo', sites), null)
})

test('effective preview prefixes use the Web mount path', () => {
  assert.deepEqual(effectiveWebMountPrefixes('/fixed/'), {
    matchPathPrefix: '/fixed', routePathPrefix: '/fixed',
  })
  assert.deepEqual(effectiveWebMountPrefixes('/'), {
    matchPathPrefix: '/', routePathPrefix: '',
  })
})
