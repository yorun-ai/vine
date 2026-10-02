import JSON5 from 'json5'
import { isMap, isScalar, isSeq, parseDocument } from 'yaml'
import { configJsonLanguage } from './config-json-language.ts'
import { configDataFields, configTypeText } from './config-structured-schema.ts'
import type { ConfigSchemaType } from './config-structured-schema.ts'
import type { ConfigJsonField } from './config-json-document.ts'

export interface ConfigSchemaHint { from: number; to: number; text: string; skelName?: string }

export function configSchemaHints(doc: string, fields: ReadonlyArray<ConfigJsonField>, yaml = false): ConfigSchemaHint[] {
  const hints: ConfigSchemaHint[] = []
  const data = fields[0]?.dataTypes ?? []
  const add = (from: number, to: number, field: ConfigJsonField) => {
    const type = field.valueType
    const enumType = type?.kind === 'list' ? type.element : type
    const options = enumType?.kind === 'enum' ? enumType.enumItems : []
    hints.push({ from, to,
      text: [type ? configTypeText(type) : field.type, field.description,
        field.sensitive ? '@sensitive' : '', field.deprecated ? `@deprecated ${field.deprecatedReason ?? ''}` : '', field.example ? `@example ${field.example}` : '',
        ...options.map((item) => `${item.name}${item.description ? ': ' + item.description : ''}`),
      ].filter(Boolean).join('\n'),
      skelName: type?.kind === 'data' || type?.kind === 'enum' ? type.name : undefined,
    })
  }
  if (yaml) {
    const document = parseDocument(doc)
    const walk = (node: unknown, type: ConfigSchemaType | null, members?: ReadonlyArray<ConfigJsonField>) => {
      if (isSeq(node) && type?.kind === 'list') for (const item of node.items) walk(item, type.element)
      if (!isMap(node)) return
      const definitions = members ?? (type?.kind === 'data' ? configDataFields(type, data) : null)
      for (const pair of node.items) {
        if (!isScalar(pair.key)) continue
        const name = String(pair.key.value)
        const field = definitions?.find((item) => item.name === name)
        if (field) {
          if (pair.key.range) add(pair.key.range[0], pair.key.range[1], field)
          walk(pair.value, field.valueType ?? null)
        } else if (type?.kind === 'map') walk(pair.value, type.value)
      }
    }
    walk(document.contents, null, fields)
    return hints
  }
  // Traverse the syntax tree rather than interpreting the draft, so hints keep
  // working while values are incomplete or invalid.
  const tree = configJsonLanguage.parser.parse(doc)
  const walk = (node: typeof tree.topNode, type: ConfigSchemaType | null, members?: ReadonlyArray<ConfigJsonField>) => {
    if (node.name === 'SingleExpression') { if (node.firstChild) walk(node.firstChild, type, members); return }
    if (node.name === 'ArrayExpression' && type?.kind === 'list') {
      for (let child = node.firstChild; child; child = child.nextSibling) if (!['[', ']', ','].includes(child.name)) walk(child, type.element)
      return
    }
    if (node.name !== 'ObjectExpression') return
    const definitions = members ?? (type?.kind === 'data' ? configDataFields(type, data) : null)
    for (let property = node.firstChild; property; property = property.nextSibling) {
      if (property.name !== 'Property') continue
      const key = property.firstChild
      if (!key) continue
      let name: string
      try { name = Object.keys(JSON5.parse(`{${doc.slice(key.from, key.to)}: null}`))[0] } catch { continue }
      const field = definitions?.find((item) => item.name === name)
      if (field) {
        add(key.from, key.to, field)
        if (property.lastChild) walk(property.lastChild, field.valueType ?? null)
      } else if (type?.kind === 'map' && property.lastChild) walk(property.lastChild, type.value)
    }
  }
  walk(tree.topNode, null, fields)
  return hints
}

export function configSchemaCompletions(doc: string, pos: number, fields: ReadonlyArray<ConfigJsonField>, yaml = false) {
  const data = fields[0]?.dataTypes ?? []
  const definitions = (type: ConfigSchemaType | null) => type?.kind === 'data' ? configDataFields(type, data) : null
  const mapValue = (type: ConfigSchemaType | null) => type?.kind === 'map' ? type.value : null
  let members: ReadonlyArray<ConfigJsonField> | null = fields
  let kind: ConfigSchemaType | null = null
  let valuePosition = false
  if (yaml) {
    const lines = doc.slice(0, pos).split('\n')
    const current = lines.pop() ?? ''
    const indentation = current.match(/^ */)![0].length
    const parents: Array<{ indent: number; key: string }> = []
    for (const line of lines) {
      const match = /^( *)(- +)?([\w]+|"[^"\n]+"|'[^'\n]+')\s*:/.exec(line)
      if (!match) continue
      const indent = match[1].length + (match[2]?.length ?? 0)
      while (parents.length && parents.at(-1)!.indent >= indent) parents.pop()
      parents.push({ indent, key: match[3].replace(/^["']|["']$/g, '') })
    }
    while (parents.length && parents.at(-1)!.indent >= indentation) parents.pop()
    for (const parent of parents) {
      while (kind?.kind === 'list') kind = kind.element
      members = kind ? definitions(kind) : members
      const field: ConfigJsonField | undefined = members?.find((field) => field.name === parent.key)
      kind = field?.valueType ?? mapValue(kind)
      members = definitions(kind)
    }
    while (kind?.kind === 'list') kind = kind.element
    members = kind ? definitions(kind) : members
    const property = /(?:^|\s)([\w]+|"[^"\n]+"|'[^'\n]+')\s*:\s*[^:]*$/.exec(current)
    if (property) { kind = members?.find((field) => field.name === property[1].replace(/^["']|["']$/g, ''))?.valueType ?? null; valuePosition = true }
  } else {
    const tree = configJsonLanguage.parser.parse(doc)
    const walk = (node: typeof tree.topNode, type: ConfigSchemaType | null, available: ReadonlyArray<ConfigJsonField> | null) => {
      if (pos < node.from || pos > node.to) return
      if (node.name === 'SingleExpression') { if (node.firstChild) walk(node.firstChild, type, available); return }
      if (node.name === 'ArrayExpression' && type?.kind === 'list') {
        kind = type.element; members = definitions(kind); valuePosition = true
        for (let child = node.firstChild; child; child = child.nextSibling) if (!['[', ']', ','].includes(child.name)) walk(child, kind, members)
      }
      if (node.name !== 'ObjectExpression') return
      kind = type; members = available ?? definitions(type); valuePosition = false
      for (let property = node.firstChild; property; property = property.nextSibling) {
        if (property.name !== 'Property' || pos < property.from || pos > property.to) continue
        const key = property.firstChild
        if (!key || pos <= key.to) continue
        const colon = key.nextSibling
        if (colon?.name !== ':' || pos < colon.to) continue
        let name: string
        try { name = Object.keys(JSON5.parse(`{${doc.slice(key.from, key.to)}: null}`))[0] } catch { continue }
        const valueType = members?.find((field) => field.name === name)?.valueType ?? (type?.kind === 'map' ? type.value : null)
        kind = valueType; valuePosition = true
        if (property.lastChild) walk(property.lastChild, valueType, definitions(valueType))
        return
      }
    }
    walk(tree.topNode, null, fields)
  }
  if (valuePosition) {
    if (kind?.kind === 'enum') return kind.enumItems.map((item) => ({ label: item.name, detail: item.description, apply: yaml ? item.name : JSON.stringify(item.name), type: 'enum' }))
    if (kind?.kind === 'scalar' && kind.name === 'bool') return ['true', 'false'].map((label) => ({ label, apply: label, type: 'keyword', detail: 'bool' }))
    return []
  }
  return (members ?? []).map((field) => ({ label: field.name, detail: `${field.valueType ? configTypeText(field.valueType) : field.type}${field.description ? ': ' + field.description : ''}`, apply: yaml ? field.name : JSON.stringify(field.name), type: 'property' }))
}
