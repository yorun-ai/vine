// Named definitions remain separate so recursive and generic data stay finite.
export interface ConfigSchemaType {
  kind: string
  nullable: boolean
  name: string
  typeArguments: ReadonlyArray<ConfigSchemaType>
  element: ConfigSchemaType | null
  key: ConfigSchemaType | null
  value: ConfigSchemaType | null
  enumItems: ReadonlyArray<{ name: string; description: string }>
}

export interface ConfigSchemaField {
  name: string
  type: string
  description: string
  valueType?: ConfigSchemaType | null
  sensitive?: boolean
  example?: string
  deprecated?: boolean
  deprecatedReason?: string
}

export interface ConfigSchemaData {
  skelName: string
  sensitive: boolean
  typeParameters: ReadonlyArray<string>
  fields: ReadonlyArray<ConfigSchemaField>
}

export function bindConfigType(type: ConfigSchemaType, bindings: ReadonlyMap<string, ConfigSchemaType>): ConfigSchemaType | null {
  if (type.kind === 'typeParameter') {
    const bound = bindings.get(type.name)
    return bound ? { ...bound, nullable: bound.nullable || type.nullable } : null
  }
  return {
    ...type,
    typeArguments: type.typeArguments.map((arg) => bindConfigType(arg, bindings)!).filter(Boolean),
    element: type.element ? bindConfigType(type.element, bindings) : null,
    key: type.key ? bindConfigType(type.key, bindings) : null,
    value: type.value ? bindConfigType(type.value, bindings) : null,
  }
}

export function configDataFields(type: ConfigSchemaType, data: ReadonlyArray<ConfigSchemaData>) {
  const definition = data.find((item) => item.skelName === type.name)
  if (!definition || definition.typeParameters.length !== type.typeArguments.length) return null
  const bindings = new Map(definition.typeParameters.map((name, index) => [name, type.typeArguments[index]]))
  return definition.fields.map((field) => ({
    ...field,
    sensitive: definition.sensitive || field.sensitive,
    valueType: field.valueType ? bindConfigType(field.valueType, bindings) : null,
  }))
}

export function configTypeText(type: ConfigSchemaType): string {
  let text = type.name
  if (type.kind === 'list') text = `list<${type.element ? configTypeText(type.element) : '?'}>`
  if (type.kind === 'map') text = `map<${type.key ? configTypeText(type.key) : '?'}, ${type.value ? configTypeText(type.value) : '?'}>`
  if (type.typeArguments.length) text += `<${type.typeArguments.map(configTypeText).join(', ')}>`
  return text + (type.nullable ? '?' : '')
}

export function defaultStructuredConfigValue(type: ConfigSchemaType, data: ReadonlyArray<ConfigSchemaData>, ancestors: ReadonlySet<string> = new Set()): unknown {
  if (type.nullable) return null
  if (type.kind === 'list') return []
  if (type.kind === 'map') return {}
  if (type.kind === 'enum') return type.enumItems[0]?.name ?? ''
  if (type.kind === 'data') {
    // Required recursive declarations are invalid Skel; keep an incomplete draft
    // finite if a registered legacy schema nevertheless contains one.
    if (ancestors.has(configTypeText(type))) return {}
    const next = new Set(ancestors).add(configTypeText(type))
    return Object.fromEntries((configDataFields(type, data) ?? []).map((field) => [field.name,
      field.valueType ? defaultStructuredConfigValue(field.valueType, data, next) : null]))
  }
  switch (type.name) {
    case 'bool': return false
    case 'int': case 'long': case 'float': case 'double': return 0
    case 'decimal': return '0'
    case 'json': return '{}'
    case 'duration': return '0s'
    case 'uuid': return '00000000-0000-0000-0000-000000000000'
    case 'timestamp': return '1970-01-01T00:00:00Z'
    case 'localdate': return '1970-01-01'
    case 'localtime': return '00:00:00'
    case 'localdatetime': return '1970-01-01T00:00:00'
    default: return ''
  }
}

export function validConfigBinary(value: unknown) {
  if (typeof value !== 'string') return false
  const text = value.replace(/[\r\n]/g, '')
  return /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(text)
}
