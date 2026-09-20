import { readFile, readdir } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

export const dashboardLicenseMarker = '=== Hub Dashboard third-party licenses ===\n'

export function cssPackageNames(styles) {
  const packages = new Set()
  const imports = styles.replace(/\/\*[\s\S]*?\*\//g, '').matchAll(/@import\s+(?:url\(\s*)?['"]([^'"]+)['"]/g)
  for (const [, specifier] of imports) {
    if (specifier.startsWith('.') || specifier.startsWith('/') || specifier.includes(':')) continue
    packages.add(specifier.split('/').slice(0, specifier.startsWith('@') ? 2 : 1).join('/'))
  }
  return [...packages].sort()
}

export async function collectDashboardLicenses(entries, dashboardDirectory) {
  const licenses = new Map(entries.map(entry => [`${entry.name}@${entry.version}`, { ...entry }]))
  // Vite's report covers bundled JS modules. CSS imports and their fonts are
  // inlined separately, so collect the packages imported by our stylesheet too.
  const styles = await readFile(join(dashboardDirectory, 'src/styles.css'), 'utf8')
  for (const name of cssPackageNames(styles)) {
    const directory = join(dashboardDirectory, 'node_modules', name)
    const pkg = JSON.parse(await readFile(join(directory, 'package.json'), 'utf8'))
    const files = (await readdir(directory)).filter(file => /^(license|licence|copying|notice)(\.|$)/i.test(file)).sort()
    const texts = await Promise.all(files.map(file => readFile(join(directory, file), 'utf8')))
    licenses.set(`${pkg.name}@${pkg.version}`, {
      name: pkg.name, version: pkg.version, identifier: pkg.license, text: texts.join('\n\n'),
    })
  }
  // These uiw packages omit LICENSE in their npm tarballs. Use the checked-in
  // upstream notice already maintained by the Dashboard.
  for (const entry of licenses.values()) {
    if (!entry.text && ['@uiw/react-codemirror', '@uiw/codemirror-extensions-basic-setup'].includes(entry.name)) {
      if (entry.identifier !== 'MIT') throw new Error(`Review changed license for ${entry.name}`)
      entry.text = await readFile(join(dashboardDirectory, 'licenses/uiw-react-codemirror-MIT.txt'), 'utf8')
    }
  }
  return [...licenses.values()]
}

export function formatDashboardLicenses(entries) {
  if (entries.length === 0) throw new Error('Dashboard dependency license inventory is empty')
  const sorted = [...entries].sort((a, b) => {
    const left = `${a.name}@${a.version}`
    const right = `${b.name}@${b.version}`
    return left < right ? -1 : left > right ? 1 : 0
  })
  return '\nIncludes bundled JavaScript dependencies and imported CSS/font packages.\n' + sorted.map(entry => {
    if (!entry.name || !entry.version || !entry.identifier || !entry.text?.trim()) {
      throw new Error(`Missing license metadata or text for ${entry.name ?? 'unknown dependency'}`)
    }
    return `\n-------------------------------------------------------------------------------\n${entry.name}@${entry.version} (${entry.identifier})\n-------------------------------------------------------------------------------\n${entry.text.replace(/\r\n/g, '\n').trim()}\n`
  }).join('')
}

export function checkDashboardLicenses(inventory, expected) {
  const sections = inventory.split(dashboardLicenseMarker)
  if (sections.length !== 2 || sections[1] !== expected) {
    throw new Error('Dashboard licenses are missing or stale; run bash script/gen-third-party-licenses.sh and include THIRD_PARTY_LICENSES.txt in the change.')
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const [mode, generated, inventory] = process.argv.slice(2)
  if (mode !== '--check' || !generated || !inventory) {
    throw new Error('Usage: dashboard-licenses.mjs --check <generated-notices> <THIRD_PARTY_LICENSES.txt>')
  }
  checkDashboardLicenses(await readFile(inventory, 'utf8'), await readFile(generated, 'utf8'))
  console.log('Dashboard license inventory matches the build.')
}
