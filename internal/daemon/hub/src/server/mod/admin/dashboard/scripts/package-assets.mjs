import { mkdir, mkdtemp, readdir, readFile, rename, rm, stat, writeFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

// Build a complete replacement beside the destination before touching the old
// bundle. Preserve every build output file without additional compression.
export async function packageAssets(source, destination) {
  if (!(await stat(join(source, 'index.html'))).isFile()) {
    throw new Error('Dashboard build is missing index.html')
  }
  await mkdir(dirname(destination), { recursive: true })
  const staging = await mkdtemp(join(dirname(destination), '.dashboard-'))
  const files = join(staging, 'files')
  const previous = join(staging, 'previous')
  let hasPrevious = false
  let count = 0
  try {
    async function copyDirectory(input, output) {
      await mkdir(output, { recursive: true })
      for (const entry of await readdir(input, { withFileTypes: true })) {
        const inputPath = join(input, entry.name)
        const outputPath = join(output, entry.name)
        if (entry.isDirectory()) {
          await copyDirectory(inputPath, outputPath)
        } else if (entry.isFile()) {
          const content = await readFile(inputPath)
          await writeFile(outputPath, content, { flag: 'wx' })
          count++
        } else {
          throw new Error(`Unsupported Dashboard asset: ${inputPath}`)
        }
      }
    }
    await copyDirectory(source, files)
    try {
      await rename(destination, previous)
      hasPrevious = true
    } catch (error) {
      if (error.code !== 'ENOENT') throw error
    }
    try {
      await rename(files, destination)
    } catch (error) {
      if (hasPrevious) await rename(previous, destination)
      throw error
    }
  } finally {
    await rm(staging, { recursive: true, force: true })
  }
  return { count }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const [source, destination] = process.argv.slice(2)
  if (!source || !destination) throw new Error('Usage: package-assets.mjs <build-directory> <assets-directory>')
  const { count } = await packageAssets(resolve(source), resolve(destination))
  console.log(`Packaged ${count} Dashboard files: ${destination}`)
}
