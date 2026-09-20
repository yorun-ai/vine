import JSON5 from 'json5'
import { isMap, isScalar, parseDocument } from 'yaml'
import { configJsonLanguage } from './config-json-language.ts'
import type { ConfigJsonField, ConfigValueRange } from './config-json-document.ts'

export function configMapTypes(type: string) {
  const match = /^map<([^,<>]+),\s*([^<>]+)>\??$/.exec(type)
  return match ? { key: match[1].trim(), value: match[2].trim() } : null
}

export function configMapEnums(field: ConfigJsonField) {
  const types = configMapTypes(field.type)
  return {
    key: field.mapKeyEnumItems ?? (types?.key.includes('.') ? field.enumItems : []) ?? [],
    value: field.mapValueEnumItems ?? [],
  }
}

export interface ConfigMapEntry {
  key: string
  keyRange: { from: number; to: number }
  valueRange: { from: number; to: number }
}

export function configMapEntries(doc: string, range: ConfigValueRange, yaml: boolean): Array<ConfigMapEntry> {
  const text = doc.slice(range.from, range.to)
  const entries: Array<ConfigMapEntry> = []
  const absolute = (from: number, to: number) => ({ from: range.from + from, to: range.from + to })
  if (yaml) {
    const document = parseDocument(text)
    if (document.errors.length || !isMap(document.contents)) {
      return entries
    }
    for (const pair of document.contents.items) {
      if (!isScalar(pair.key) || pair.key.value === null || !pair.key.range || !pair.value?.range) {
        continue
      }
      const from = pair.value.range[0]
      const to = from + text.slice(from, pair.value.range[1]).trimEnd().length
      entries.push({ key: typeof pair.key.value === 'string' ? pair.key.value : pair.key.source ?? String(pair.key.value), keyRange: absolute(pair.key.range[0], pair.key.range[1]), valueRange: absolute(from, to) })
    }
    return entries
  }
  configJsonLanguage.parser.parse(text).iterate({
    enter(node) {
      if (node.name !== 'Property' || node.node.parent?.parent?.name !== 'SingleExpression') {
        return
      }
      const key = node.node.firstChild
      const value = node.node.lastChild
      if (!key || !value || key === value) {
        return
      }
      try {
        const name = Object.keys(JSON5.parse(`{${text.slice(key.from, key.to)}: null}`))[0]
        entries.push({ key: name, keyRange: absolute(key.from, key.to), valueRange: absolute(value.from, value.to) })
      } catch {
        return
      }
    },
  })
  return entries
}

export function configMapEnumIssues(value: unknown, field: ConfigJsonField) {
  if (!configMapTypes(field.type) || value === null || typeof value !== 'object' || Array.isArray(value)) {
    return []
  }
  const enums = configMapEnums(field)
  return Object.entries(value).flatMap(([key, item]) => {
    const issues: Array<{ key: string; part: 'key' | 'value'; expected: string; actual: string }> = []
    if (enums.key.length && !enums.key.some((option) => option.name === key)) {
      issues.push({ key, part: 'key', expected: enums.key.map((option) => option.name).join(' | '), actual: JSON.stringify(key) })
    }
    if (enums.value.length && !(item === null && configMapTypes(field.type)!.value.endsWith('?')) && !enums.value.some((option) => option.name === item)) {
      issues.push({ key, part: 'value', expected: enums.value.map((option) => option.name).join(' | '), actual: JSON.stringify(item) })
    }
    return issues
  })
}
