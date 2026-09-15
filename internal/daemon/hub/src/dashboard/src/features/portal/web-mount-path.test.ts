import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { lockWebMountPath, lockedWebMountPath } from './web-mount-path.ts'

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

test('applies the mount path to both prefixes', () => {
  const value = {
    name: 'rule',
    matchPathPrefix: '/other',
    routePathPrefix: '',
  }

  assert.deepEqual(lockWebMountPath(value, '/demo'), {
    name: 'rule',
    matchPathPrefix: '/demo',
    routePathPrefix: '/demo',
  })
  assert.equal(lockWebMountPath(value, null), value)
  const locked = lockWebMountPath(value, '/demo')
  assert.equal(lockWebMountPath(locked, '/demo'), locked)
})
