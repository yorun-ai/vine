import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { checkDashboardLicenses, collectDashboardLicenses, cssPackageNames, dashboardLicenseMarker, formatDashboardLicenses } from './dashboard-licenses.mjs'

test('extracts stylesheet packages, including fonts, without local files or URLs', () => {
  assert.deepEqual(cssPackageNames(`
    @import 'tailwindcss';
    @import 'tw-animate-css';
    @import 'shadcn/tailwind.css';
    @import '@fontsource-variable/geist';
    @import url('@fontsource-variable/geist/latin.css');
    @import './local.css';
    @import '/absolute.css';
    @import 'https://example.com/styles.css';
    /* @import 'unused-package'; */
  `), ['@fontsource-variable/geist', 'shadcn', 'tailwindcss', 'tw-animate-css'])
})

test('formats complete notices deterministically and rejects missing texts', () => {
  const entries = [
    { name: 'z', version: '1', identifier: 'MIT', text: 'Copyright Z\r\nMIT text\r\n' },
    { name: 'a', version: '2', identifier: 'OFL-1.1', text: 'Copyright A\nFont license\n' },
  ]
  const text = formatDashboardLicenses(entries)
  assert.equal(text, formatDashboardLicenses([...entries].reverse()))
  assert.ok(text.indexOf('a@2') < text.indexOf('z@1'))
  assert.ok(text.includes('Copyright Z\nMIT text'))
  assert.ok(text.includes('Copyright A\nFont license'))
  assert.throws(() => formatDashboardLicenses([{ name: 'missing', version: '1', identifier: 'MIT' }]), /Missing license/)
  assert.throws(() => formatDashboardLicenses([]), /empty/)
})

test('checks only the Dashboard section and rejects missing, stale or duplicate sections', () => {
  const text = formatDashboardLicenses([{ name: 'react', version: '19', identifier: 'MIT', text: 'React license' }])
  assert.doesNotThrow(() => checkDashboardLicenses(`Go notices\n${dashboardLicenseMarker}${text}`, text))
  for (const inventory of ['Go notices only', `Go\n${dashboardLicenseMarker}stale`, dashboardLicenseMarker.repeat(2) + text]) {
    assert.throws(() => checkDashboardLicenses(inventory, text), /gen-third-party-licenses.sh/)
  }
})

test('includes CSS and font license texts and fills missing uiw notices', async (t) => {
  const directory = await mkdtemp(join(tmpdir(), 'vine-license-test-'))
  t.after(() => rm(directory, { recursive: true, force: true }))
  const files = {
    'src/styles.css': "@import '@fontsource-variable/geist';",
    'node_modules/@fontsource-variable/geist/package.json': JSON.stringify({ name: '@fontsource-variable/geist', version: '1.0', license: 'OFL-1.1' }),
    'node_modules/@fontsource-variable/geist/LICENSE': 'Geist font license',
    'node_modules/@fontsource-variable/geist/NOTICE': 'Font attribution',
    'licenses/uiw-react-codemirror-MIT.txt': 'uiw MIT license',
  }
  for (const [filename, content] of Object.entries(files)) {
    await mkdir(dirname(join(directory, filename)), { recursive: true })
    await writeFile(join(directory, filename), content)
  }
  const entry = { name: '@uiw/react-codemirror', version: '4.25', identifier: 'MIT' }
  const entries = await collectDashboardLicenses([entry], directory)
  const text = formatDashboardLicenses(entries)
  assert.ok(text.includes('@fontsource-variable/geist@1.0 (OFL-1.1)'))
  assert.ok(text.includes('Geist font license\n\nFont attribution'))
  assert.ok(text.includes(await readFile(join(directory, 'licenses/uiw-react-codemirror-MIT.txt'), 'utf8')))
  assert.equal(entry.text, undefined)
  await assert.rejects(collectDashboardLicenses([{ ...entry, identifier: 'changed' }], directory), /Review changed license/)
})
