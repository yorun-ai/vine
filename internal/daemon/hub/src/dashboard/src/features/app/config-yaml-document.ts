import { isAlias, isNode, isMap, isScalar, parseDocument, stringify, visit } from 'yaml'
import { normalizeConfigJson } from './config-json-document.ts'
import type { ConfigJsonField, ConfigValueRange, createConfigJsonDocument } from './config-json-document.ts'

export function parseConfigYaml(text: string): unknown {
  const document = parseDocument(text, { version: '1.2', uniqueKeys: true })
  const problem = document.errors[0] ?? document.warnings[0]
  if (problem) {
    throw problem
  }
  visit(document, (_key, node) => {
    if (isScalar(node) && node.type === 'PLAIN' && node.tag !== 'tag:yaml.org,2002:str') {
      const source = node.source ?? ''
      const numeric = typeof node.value === 'number' || /^[+-]?(?:(?:[0-9][0-9_]*(?:\.[0-9_]*)?|\.[0-9_]+)(?:[eE][+-]?[0-9_]+)?|0[xX][0-9a-fA-F_]+|0[oO][0-7_]+|0[bB][01_]+)$/.test(source)
      if (numeric && !/^[+-]?(0|[1-9][0-9]*)(\.[0-9]+)?$/.test(source)) {
        throw Object.assign(new Error('Unsupported YAML number; use plain decimal notation without separators, leading zeros, or exponents.'), { pos: node.range })
      }
    }
    if (isAlias(node) || (isNode(node) && 'anchor' in node && node.anchor)) {
      throw Object.assign(new Error('YAML anchors and aliases are not supported.'), { pos: node.range })
    }
  })
  visit(document, {
    Scalar(_key, scalar) {
      if (typeof scalar.value === 'number' && !Number.isFinite(scalar.value)) {
        throw Object.assign(new Error('Configuration numbers must be finite.'), { pos: scalar.range })
      }
      if (typeof scalar.value === 'number' && Number.isInteger(scalar.value) && !Number.isSafeInteger(scalar.value)) {
        throw Object.assign(new Error('Configuration integers must be within the safe range: -9007199254740991 to 9007199254740991.'), { pos: scalar.range })
      }
    },
    Map(_key, map) {
      const keys = new Set<string>()
      for (const pair of map.items) {
        const key = pair.key
        if (!isScalar(key) || key.value === null) {
          throw Object.assign(new Error('Configuration map keys must be non-null scalars.'), { pos: isScalar(key) ? key.range : undefined })
        }
        if ((key.type === 'PLAIN' && key.source === '<<') || key.tag === 'tag:yaml.org,2002:merge') {
          throw Object.assign(new Error('YAML merge keys are not supported.'), { pos: key.range })
        }
        const name = typeof key.value === 'string' ? key.value : key.source ?? String(key.value)
        if (keys.has(name)) {
          throw Object.assign(new Error(`Duplicate configuration map key: ${name}`), { pos: key.range })
        }
        keys.add(name)
        key.value = name
      }
    },
  })
  const value: unknown = document.toJS({ maxAliasCount: 100 })
  JSON.stringify(value, (_key, item: unknown) => {
    if (typeof item === 'number' && !Number.isFinite(item)) {
      throw new Error('Configuration numbers must be finite.')
    }
    return item
  })
  return value
}

export function normalizeConfigYaml(text: string) {
  return JSON.stringify(parseConfigYaml(text), null, 2)
}

function stringifyConfigYaml(value: unknown) {
  return stringify(value, {
    lineWidth: 0,
    customTags: (tags) => tags.map((tag) => typeof tag !== 'string' && tag.collection === undefined && tag.tag === 'tag:yaml.org,2002:float' ? {
      ...tag,
      stringify: (node: { value: unknown }) => {
        const text = String(node.value)
        const match = /^(-?)(\d+)(?:\.(\d+))?e([+-]?\d+)$/i.exec(text)
        if (!match) {
          return text
        }
        const digits = match[2] + (match[3] ?? '')
        const point = match[2].length + Number(match[4])
        const decimal = point <= 0 ? `0.${'0'.repeat(-point)}${digits}`
          : point >= digits.length ? digits + '0'.repeat(point - digits.length)
          : `${digits.slice(0, point)}.${digits.slice(point)}`
        return match[1] + decimal
      },
    } : tag),
  })
}

export function formatConfigYaml(value: string) {
  return stringifyConfigYaml(JSON.parse(normalizeConfigJson(value)))
}

export function getConfigYamlPropertyRanges(text: string): Array<ConfigValueRange> {
  const document = parseDocument(text)
  if (!isMap(document.contents)) {
    return []
  }
  return document.contents.items.flatMap((pair) => {
    if (!isScalar(pair.key) || typeof pair.key.value !== 'string' || !pair.value?.range) {
      return []
    }
    return [{ name: pair.key.value, from: pair.value.range[0], to: pair.value.range[1] }]
  })
}

export function getConfigYamlErrors(text: string, ranges: ReadonlyArray<ConfigValueRange> = []) {
  try {
    const value = parseConfigYaml(text)
    if (ranges.length > 0) {
      const keys = value && typeof value === 'object' && !Array.isArray(value) ? Object.keys(value) : []
      if (keys.length !== ranges.length || keys.some((key) => !ranges.some((range) => range.name === key))) {
        throw new Error('Configuration keys are read-only; use Replace to change the complete configuration.')
      }
    }
    return []
  } catch (error) {
    const problem = error as Error & { pos?: [number, number] }
    const from = Math.max(0, Math.min(problem.pos?.[0] ?? 0, text.length - 1))
    const to = Math.min(Math.max(from + 1, problem.pos?.[1] ?? from + 1), text.length)
    const field = ranges.find((range) => from >= range.from && from <= range.to)
    return [{ name: field?.name ?? '', from, to, message: problem.message }]
  }
}

export function createConfigYamlDocument(value: string, fields: ReadonlyArray<ConfigJsonField>, lockKeys: boolean): ReturnType<typeof createConfigJsonDocument> {
  if (!lockKeys) {
    let doc = value
    try {
      doc = formatConfigYaml(value)
    } catch {
      // Preserve malformed replacements for correction.
    }
    return { lockKeys, doc, ranges: [{ name: '', from: 0, to: doc.length }], typeRanges: [] as Array<{ fieldName: string; typeName: string; from: number; to: number }> }
  }
  const ranges: Array<ConfigValueRange> = []
  const typeRanges: Array<{ fieldName: string; typeName: string; from: number; to: number }> = []
  let doc = ''
  for (const [name, item] of Object.entries(JSON.parse(value) as Record<string, unknown>)) {
    const field = fields.find((field) => field.name === name)
    const blockFrom = doc.length
    for (const match of (field?.type ?? '').matchAll(/[A-Za-z_][A-Za-z0-9_.]*/g)) {
      const from = doc.length + 2 + match.index
      typeRanges.push({ fieldName: name, typeName: match[0], from, to: from + match[0].length })
    }
    for (const line of [field?.type, field?.description].filter(Boolean).join('\n').split(/\r\n|[\n\r\u2028\u2029]/)) {
      doc += `# ${line}\n`
    }
    const entry = stringifyConfigYaml({ [name]: item })
    const pair = (parseDocument(entry).contents as import('yaml').YAMLMap).items[0]
    const start = pair.key && isScalar(pair.key) ? pair.key.range![1] : 0
    const from = doc.length + start + 1
    doc += entry.trimEnd()
    ranges.push({ name, from, to: doc.length, blockFrom })
    doc += '\n\n'
  }
  return { lockKeys, doc: doc || '{}\n', ranges, typeRanges }
}
