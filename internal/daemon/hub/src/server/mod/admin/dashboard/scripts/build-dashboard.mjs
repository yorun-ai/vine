import { readFile, readdir, rm, rmdir, writeFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { build } from 'vite'
import { collectDashboardLicenses, formatDashboardLicenses } from './dashboard-licenses.mjs'

const [output, notices] = process.argv.slice(2)
if (!output || !notices) throw new Error('Usage: build-dashboard.mjs <temporary-dist> <temporary-notices>')
const dashboardDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const outDir = resolve(output)
const report = '.vite/dependency-licenses.json'
await build({
  root: dashboardDirectory,
  configFile: join(dashboardDirectory, 'vite.config.ts'),
  build: { outDir, emptyOutDir: true, license: { fileName: report } },
})
const entries = JSON.parse(await readFile(join(outDir, report), 'utf8'))
const licenses = await collectDashboardLicenses(entries, dashboardDirectory)
await writeFile(resolve(notices), formatDashboardLicenses(licenses))
// License metadata is only a temporary build input, never a Dashboard asset.
await rm(join(outDir, report))
if ((await readdir(join(outDir, '.vite'))).length === 0) await rmdir(join(outDir, '.vite'))
console.log(`Collected licenses for ${licenses.length} Dashboard dependencies.`)
