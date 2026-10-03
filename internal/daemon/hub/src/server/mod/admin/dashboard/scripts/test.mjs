import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { discoverTests } from './test-files.mjs'

const root = fileURLToPath(new URL('../', import.meta.url))
const tests = discoverTests(root)
if (tests.length === 0) throw new Error('No Dashboard unit tests found')
const result = spawnSync(process.execPath, ['--experimental-strip-types', '--test', ...process.argv.slice(2), ...tests], {
  cwd: root,
  stdio: 'inherit',
})
if (result.error) throw result.error
process.exitCode = result.status ?? 1
