import { configDataFields, configTypeText, validConfigBinary } from './config-structured-descriptor.ts'
import type { ConfigDescriptorType, ConfigDescriptorData } from './config-structured-descriptor.ts'
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

export function configValueIssues(value: unknown, field: ConfigJsonField, data: ReadonlyArray<ConfigDescriptorData> = field.dataTypes ?? []): ConfigValueIssue[] {
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
      case 'long':
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
      case 'binary':
        valid = validConfigBinary(item)
        break
      case 'bool':
        valid = typeof item === 'boolean'
        break
      case 'json':
        valid = typeof item === 'string'
        break
      default:
        valid = typeof item === 'string'
    }
    if (!valid) {
      issues.push({ path, part: 'value', expected: base, actual: typeof item === 'string' ? JSON.stringify(item) : describe(item) })
    }
  }
  function sensitiveType(type: ConfigDescriptorType | null): boolean {
    if (type?.kind === 'data') return !!data.find((declaration) => declaration.skelName === type.name)?.sensitive
    if (type?.kind === 'list') return sensitiveType(type.element)
    if (type?.kind === 'map') return sensitiveType(type.value)
    return false
  }
  function structured(item: unknown, type: ConfigDescriptorType | null, path: string, sensitive: boolean) {
    const fail = (expected = type ? configTypeText(type) : 'unresolved type', actual = describe(item)) => issues.push({ path, part: 'value', expected, actual })
    if (!type) { fail(); return }
    sensitive ||= sensitiveType(type)
    if (item === null) { if (!type.nullable) fail(); return }
    if (type.kind === 'data') {
      if (typeof item !== 'object' || Array.isArray(item)) { fail(); return }
      const fields = configDataFields(type, data)
      if (!fields) { fail(); return }
      const object = item as Record<string, unknown>
      for (const member of fields) {
        const memberPath = `${path}.${member.name}`
        if (!Object.hasOwn(object, member.name)) {
          issues.push({ path: memberPath, part: 'value', expected: member.valueType ? configTypeText(member.valueType) : member.type, actual: 'missing' })
        } else structured(object[member.name], member.valueType, memberPath, sensitive || !!member.sensitive)
      }
      for (const key of Object.keys(object)) {
        if (!fields.some((member) => member.name === key)) {
          issues.push({ path: sensitive ? path : `${path}[${JSON.stringify(key)}]`, part: 'value', expected: 'declared field', actual: 'unknown field' })
        }
      }
      return
    }
    if (type.kind === 'list') {
      if (!Array.isArray(item)) { fail(); return }
      item.forEach((entry, index) => structured(entry, type.element, `${path}[${index}]`, sensitive))
      return
    }
    if (type.kind === 'map') {
      if (typeof item !== 'object' || Array.isArray(item)) { fail(); return }
      for (const [key, entry] of Object.entries(item)) {
        const entryPath = sensitive ? `${path}[*]` : `${path}[${JSON.stringify(key)}]`
        const kind = type.key
        const validKey = kind?.kind === 'enum' ? kind.enumItems.some((option) => option.name === key)
          : kind?.name === 'int' || kind?.name === 'long' ? /^-?(0|[1-9]\d*)$/.test(key) && BigInt(key) >= -(1n << 63n) && BigInt(key) < (1n << 63n)
          : kind?.name === 'uuid' ? validConfigUUID(key) : kind?.kind === 'scalar' && kind.name === 'string'
        if (!validKey) issues.push({ path: entryPath, part: 'key', expected: kind ? configTypeText(kind) : 'unresolved type', actual: sensitive ? 'string' : JSON.stringify(key) })
        structured(entry, type.value, entryPath, sensitive)
      }
      return
    }
    if (type.kind === 'enum') {
      if (!type.enumItems.some((option) => option.name === item)) fail(type.enumItems.map((option) => option.name).join(' | ') || type.name)
      return
    }
    if (type.kind !== 'scalar') { fail(); return }
    if (type.name === 'binary') { if (!validConfigBinary(item)) fail('binary (base64)'); return }
    // Reuse scalar format validation, but never include sensitive values in diagnostics.
    const start = issues.length
    check(item, type.name, [], path)
    if (sensitive) for (const issue of issues.slice(start)) issue.actual = describe(item)
  }
  if (field.valueType) structured(value, field.valueType, field.name, !!field.sensitive)
  else check(value, field.type, field.enumItems ?? [], field.name)
  return issues
}
