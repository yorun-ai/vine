import { configScalarFormatMatches, validConfigUUID } from './config-scalar-validation.ts'
import type { ConfigJsonField } from './config-json-document.ts'
import { configMapEnums, configMapTypes } from './config-map-enum.ts'

type EnumItems = ReadonlyArray<{ name: string }>

export interface ConfigValueIssue {
  path: string
  part: 'key' | 'value'
  expected: string
  actual: string
}

export function configValueIssues(value: unknown, field: ConfigJsonField): ConfigValueIssue[] {
  const issues: ConfigValueIssue[] = []
  const describe = (item: unknown) => item === null ? 'null' : Array.isArray(item) ? 'list' : typeof item
  function check(item: unknown, type: string, enums: EnumItems, path: string) {
    if (item === null && type.endsWith('?')) {
      return
    }
    const base = type.replace(/\?$/, '')
    const fail = (expected = base) => issues.push({ path, part: 'value', expected, actual: describe(item) })
    const list = /^list<(.+)>$/.exec(base)
    if (list) {
      if (!Array.isArray(item)) {
        fail()
        return
      }
      item.forEach((entry, index) => check(entry, list[1], enums, `${path}[${index}]`))
      return
    }
    const map = configMapTypes(base)
    if (map) {
      if (item === null || typeof item !== 'object' || Array.isArray(item)) {
        fail()
        return
      }
      const mapEnums = configMapEnums(field)
      for (const [key, entry] of Object.entries(item)) {
        const entryPath = `${path}[${JSON.stringify(key)}]`
        let validKey = true
        if (mapEnums.key.length) {
          validKey = mapEnums.key.some((option) => option.name === key)
        } else if (map.key === 'int') {
          validKey = /^-?(0|[1-9]\d*)$/.test(key) && BigInt(key) >= -(1n << 63n) && BigInt(key) < (1n << 63n)
        } else if (map.key === 'uuid') {
          validKey = validConfigUUID(key)
        }
        if (!validKey) {
          issues.push({ path: entryPath, part: 'key', expected: mapEnums.key.map((option) => option.name).join(' | ') || map.key, actual: JSON.stringify(key) })
        }
        check(entry, map.value, mapEnums.value, entryPath)
      }
      return
    }
    if (enums.length) {
      if (!enums.some((option) => option.name === item)) {
        issues.push({ path, part: 'value', expected: enums.map((option) => option.name).join(' | '), actual: JSON.stringify(item) })
      }
      return
    }
    let valid: boolean
    switch (base) {
      case 'int':
        valid = typeof item === 'number' && Number.isSafeInteger(item)
        break
      case 'float':
      case 'double':
      case 'number':
        valid = typeof item === 'number' && Number.isFinite(item)
        break
      case 'decimal':
      case 'duration':
      case 'uuid':
      case 'timestamp':
      case 'localdate':
      case 'localtime':
      case 'localdatetime':
        valid = configScalarFormatMatches(item, base)
        break
      case 'bool':
        valid = typeof item === 'boolean'
        break
      case 'json':
        valid = item !== null
        break
      default:
        valid = typeof item === 'string'
    }
    if (!valid) {
      issues.push({ path, part: 'value', expected: base, actual: typeof item === 'string' ? JSON.stringify(item) : describe(item) })
    }
  }
  check(value, field.type, field.enumItems ?? [], field.name)
  return issues
}
