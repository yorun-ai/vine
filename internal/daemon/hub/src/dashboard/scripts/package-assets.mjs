import { mkdir, mkdtemp, readdir, readFile, rename, rm, stat, writeFile } from 'node:fs/promises'
import { dirname, extname, join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { brotliCompressSync, constants } from 'node:zlib'

const textExtensions = new Set(['.html', '.js', '.mjs', '.css', '.svg', '.json', '.txt', '.xml', '.webmanifest', '.map'])

// Build a complete replacement beside the destination before touching the old
// bundle. Each logical path has exactly one representation: original or .br.
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
  let compressedCount = 0
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
          const compressed = textExtensions.has(extname(entry.name).toLowerCase())
            ? brotliCompressSync(content, { params: { [constants.BROTLI_PARAM_QUALITY]: 11 } })
            : null
          if (compressed && compressed.length < content.length) {
            await writeFile(`${outputPath}.br`, compressed, { flag: 'wx' })
            compressedCount++
          } else {
            await writeFile(outputPath, content, { flag: 'wx' })
          }
          count++
        } else {
          throw new Error(`Unsupported Dashboard asset: ${inputPath}`)
        }
      }
    }
    await copyDirectory(source, files)
    // Preserve the tracked empty placeholder so release builds stay clean.
    await writeFile(join(files, ".gitkeep"), "")
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
  return { count, compressedCount }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const [source, destination] = process.argv.slice(2)
  if (!source || !destination) throw new Error('Usage: package-assets.mjs <build-directory> <assets-directory>')
  const { count, compressedCount } = await packageAssets(resolve(source), resolve(destination))
  console.log(`Packaged ${count} Dashboard files (${compressedCount} Brotli): ${destination}`)
}
