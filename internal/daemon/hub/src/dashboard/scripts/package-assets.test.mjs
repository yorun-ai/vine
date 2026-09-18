import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readdir, readFile, rm, symlink, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { brotliDecompressSync } from 'node:zlib'
import { packageAssets } from './package-assets.mjs'

async function fixture(t) {
  const directory = await mkdtemp(join(tmpdir(), 'vine-assets-test-'))
  t.after(() => rm(directory, { recursive: true, force: true }))
  const source = join(directory, 'dist')
  const destination = join(directory, 'assets', 'dashboard')
  await mkdir(join(source, 'assets'), { recursive: true })
  await mkdir(destination, { recursive: true })
  await writeFile(join(destination, 'obsolete.js'), 'old build')
  await writeFile(join(destination, '.gitkeep'), '')
  await writeFile(join(source, 'index.html'), '<html><body>dashboard</body></html>'.repeat(20))
  return { source, destination }
}

test('packages one representation per file, preserves binary assets and removes stale files', async (t) => {
  const { source, destination } = await fixture(t)
  const javascript = 'console.log("dashboard");\n'.repeat(100)
  const png = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10, ...Array(100).fill(0)])
  await writeFile(join(source, 'assets', 'app.js'), javascript)
  await writeFile(join(source, 'assets', 'tiny.txt'), 'x')
  await writeFile(join(source, 'assets', 'logo.png'), png)
  await writeFile(join(source, 'assets', 'font.woff2'), png)
  assert.deepEqual(await packageAssets(source, destination), { count: 5, compressedCount: 2 })
  assert.deepEqual((await readdir(destination)).sort(), ['.gitkeep', 'assets', 'index.html.br'])
  assert.deepEqual((await readdir(join(destination, 'assets'))).sort(), ['app.js.br', 'font.woff2', 'logo.png', 'tiny.txt'])
  assert.equal(brotliDecompressSync(await readFile(join(destination, 'assets', 'app.js.br'))).toString(), javascript)
  assert.deepEqual(await readFile(join(destination, 'assets', 'logo.png')), png)
  assert.deepEqual(await readFile(join(destination, 'assets', 'font.woff2')), png)
  assert.equal(await readFile(join(destination, 'assets', 'tiny.txt'), 'utf8'), 'x')

  // A later build with different hashed names replaces the complete bundle.
  await rm(join(source, 'assets', 'app.js'))
  await writeFile(join(source, 'assets', 'app-next.js'), javascript)
  await packageAssets(source, destination)
  assert.equal((await readdir(join(destination, 'assets'))).includes('app.js.br'), false)
  assert.deepEqual(await readdir(join(destination, '..')), ['dashboard'])
  assert.equal(await readFile(join(destination, '.gitkeep'), 'utf8'), '')
})

test('a packaging failure preserves the previous bundle and cleans staging', async (t) => {
  const { source, destination } = await fixture(t)
  await symlink(join(source, 'index.html'), join(source, 'assets', 'link.html'))
  await assert.rejects(packageAssets(source, destination), /Unsupported Dashboard asset/)
  assert.equal(await readFile(join(destination, 'obsolete.js'), 'utf8'), 'old build')
  assert.deepEqual(await readdir(join(destination, '..')), ['dashboard'])
  assert.equal(await readFile(join(destination, '.gitkeep'), 'utf8'), '')
})

test('rejects an incomplete frontend build before replacing existing assets', async (t) => {
  const { source, destination } = await fixture(t)
  await rm(join(source, 'index.html'))
  await assert.rejects(packageAssets(source, destination), { code: 'ENOENT' })
  assert.equal(await readFile(join(destination, 'obsolete.js'), 'utf8'), 'old build')
})
