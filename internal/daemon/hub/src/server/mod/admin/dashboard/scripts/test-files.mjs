import { readdirSync } from 'node:fs'
import { join, relative } from 'node:path'

export function discoverTests(root) {
  const tests = []
  function visit(directory) {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const path = join(directory, entry.name)
      if (entry.isDirectory()) visit(path)
      else if (/\.test\.(ts|mjs)$/.test(entry.name)) tests.push(relative(root, path))
    }
  }
  for (const directory of ['scripts', 'src']) visit(join(root, directory))
  return tests.sort()
}
