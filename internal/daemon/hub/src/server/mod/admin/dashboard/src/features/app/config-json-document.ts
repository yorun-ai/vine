import JSON5 from 'json5'

export interface ConfigJsonField {
  name: string
  type: string
  description: string
  commentTags?: ReadonlyArray<string>
  sourceComment?: string
  enumItems?: ReadonlyArray<{ name: string; description: string }>
  mapKeyEnumItems?: ReadonlyArray<{ name: string; description: string }>
  mapValueEnumItems?: ReadonlyArray<{ name: string; description: string }>
}

export interface ConfigValueRange {
  blockFrom?: number
  name: string
  from: number
  to: number
}

export interface ConfigTypeRange {
  fieldName: string
  typeName: string
  from: number
  to: number
}

export function configFieldCommentLines(field: ConfigJsonField | undefined) {
  const tags = field?.commentTags ?? ['type', 'desc']
  const lines: Array<{ tag: string; value: string }> = []
  const add = (tag: string, value: string) => {
    if (!tags.includes(tag) && tag !== 'variables' && tag !== 'template') return
    for (const [index, line] of value.split(/\r\n|[\n\r\u2028\u2029]/).entries()) {
      lines.push({ tag: index === 0 ? '@' + tag : '', value: line })
    }
  }
  if (field?.type) add('type', field.type)
  if (field?.description) add('desc', field.description)
  for (const line of field?.sourceComment?.split('\n') ?? []) {
    const match = line.match(/^@(\w+) (.*)$/)
    if (match) add(match[1], match[2])
  }
  const width = Math.max(0, ...lines.map((line) => line.tag.length))
  return lines.map((line) => `${line.tag.padEnd(width)} ${line.value}`)
}

export function createConfigJsonDocument(
  value: string,
  fields: ReadonlyArray<ConfigJsonField>,
  lockKeys = true,
) {
  if (!lockKeys) {
    return { lockKeys, doc: value, ranges: [{ name: '', from: 0, to: value.length }], typeRanges: [] as Array<ConfigTypeRange> }
  }
  const object = JSON.parse(value) as Record<string, unknown>
  const fieldIndex = new Map(fields.map((field) => [field.name, field]))
  const entries = Object.entries(object)
  const ranges: Array<ConfigValueRange> = []
  const typeRanges: Array<ConfigTypeRange> = []
  let doc = '{\n'

  for (const [index, [name, fieldValue]] of entries.entries()) {
    const blockFrom = doc.length
    const field = fieldIndex.get(name)
    const comments = configFieldCommentLines(field)
    if (comments.length > 0) {
      for (const [index, line] of comments.entries()) {
        const prefix = '  // '
        if (index === 0 && line.startsWith('@type ')) {
          const valueOffset = line.match(/^@type +/)![0].length
          for (const match of line.slice(valueOffset).matchAll(/[A-Za-z_][A-Za-z0-9_.]*/g)) {
            const from = doc.length + prefix.length + valueOffset + match.index
            typeRanges.push({ fieldName: name, typeName: match[0], from, to: from + match[0].length })
          }
        }
        doc += `${prefix}${line}\n`
      }
    }
    doc += `  ${JSON.stringify(name)}: `
    const from = doc.length
    doc += JSON.stringify(fieldValue, null, 2).replace(/\n/g, '\n  ')
    ranges.push({ name, from, to: doc.length, blockFrom })
    doc += index < entries.length - 1 ? ',\n\n' : '\n'
  }

  return { lockKeys, doc: `${doc}}`, ranges, typeRanges }
}

function parseConfigJsonValue(value: string): unknown {
  return JSON5.parse(value, (_key, item: unknown) => {
    if (typeof item === 'number' && !Number.isFinite(item)) {
      throw new Error('Configuration numbers must be finite.')
    }
    if (typeof item === 'number' && Number.isInteger(item) && !Number.isSafeInteger(item)) {
      throw new Error('Configuration integers must be within the safe range: -9007199254740991 to 9007199254740991.')
    }
    return item
  })
}

export function normalizeConfigJson(value: string) {
  return JSON.stringify(parseConfigJsonValue(value), null, 2)
}

export function getFreeConfigJsonErrors(doc: string) {
  try {
    normalizeConfigJson(doc)
    return []
  } catch (error) {
    const syntaxError = error as Error & { lineNumber?: number; columnNumber?: number }
    const offset = doc.split('\n').slice(0, (syntaxError.lineNumber ?? 1) - 1)
      .reduce((length, line) => length + line.length + 1, 0) + (syntaxError.columnNumber ?? 1) - 1
    const from = Math.max(0, Math.min(offset, doc.length - 1))
    return [{ name: '', from, to: Math.min(from + 1, doc.length), message: syntaxError.message }]
  }
}

export function extractConfigJson(doc: string, ranges: ReadonlyArray<ConfigValueRange>) {
  JSON5.parse(doc)
  return JSON.stringify(Object.fromEntries(ranges.map((range) => [
    range.name,
    parseConfigJsonValue(doc.slice(range.from, range.to)),
  ])), null, 2)
}

export function getConfigJsonErrors(doc: string, ranges: ReadonlyArray<ConfigValueRange>) {
  const errors: Array<ConfigValueRange & { message: string }> = []
  for (const range of ranges) {
    try {
      parseConfigJsonValue(doc.slice(range.from, range.to))
    } catch (error) {
      errors.push({ ...range, message: (error as Error).message })
    }
  }
  if (errors.length === 0) {
    try {
      JSON5.parse(doc)
    } catch (error) {
      const syntaxError = error as Error & { lineNumber: number; columnNumber: number }
      const lines = doc.split('\n')
      const offset = lines.slice(0, syntaxError.lineNumber - 1)
        .reduce((length, line) => length + line.length + 1, 0) + syntaxError.columnNumber - 1
      const range = ranges.find((item) => item.to >= offset) ?? ranges.at(-1)
      if (range) {
        errors.push({ ...range, message: syntaxError.message })
      }
    }
  }
  return errors
}

export function isConfigValueChange(
  ranges: ReadonlyArray<ConfigValueRange>,
  from: number,
  to: number,
) {
  return ranges.some((range) => from >= range.from && to <= range.to)
}

export function getConfigJsonTypeLinks(
  document: ReturnType<typeof createConfigJsonDocument>,
  currentRanges: ReadonlyArray<ConfigValueRange>,
  definitions: ReadonlyMap<string, { skelName: string }>,
) {
  const initialOffsets = new Map(document.ranges.map((range) => [range.name, range.from]))
  const currentOffsets = new Map(currentRanges.map((range) => [range.name, range.from]))
  return document.typeRanges.flatMap((range) => {
    const definition = definitions.get(range.typeName)
    if (!definition) {
      return []
    }
    const shift = currentOffsets.get(range.fieldName)! - initialOffsets.get(range.fieldName)!
    return [{
      from: range.from + shift,
      to: range.to + shift,
      skelName: definition.skelName,
      href: `/skeleton/data/${encodeURIComponent(definition.skelName)}`,
    }]
  })
}

export function getConfigJsonDirtyLines(
  doc: string,
  ranges: ReadonlyArray<ConfigValueRange>,
  dirtyFields: ReadonlySet<string>,
  errors: ReadonlyMap<string, string>,
) {
  const lines = new Set<number>()
  for (const range of ranges) {
    if (!dirtyFields.has(range.name) || errors.has(range.name)) {
      continue
    }
    const from = range.blockFrom ?? range.from
    let line = from === 0 ? 0 : doc.lastIndexOf('\n', from - 1) + 1
    while (line < range.to) {
      lines.add(line)
      const next = doc.indexOf('\n', line)
      if (next === -1) {
        break
      }
      line = next + 1
    }
  }
  return [...lines].sort((left, right) => left - right)
}
