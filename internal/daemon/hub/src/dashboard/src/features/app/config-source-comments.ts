interface FieldSource {
  path: string
  source: string
  define: string
  override: string
  variables: ReadonlyArray<string>
}

// Sources are JSON Pointers into the seed entity, not schema field names.
export function configSourceComment(name: string, sources: ReadonlyArray<FieldSource>) {
  const path = `/value/${name.replace(/~/g, '~0').replace(/\//g, '~1')}`
  const ancestor = sources.filter((source) => path === source.path || path.startsWith(`${source.path}/`))
    .sort((a, b) => b.path.length - a.path.length)[0]
  if (!ancestor) return '@override '
  const source = ancestor.source
  return [
    source && `@source ${source}`,
    ancestor.define && `@define ${ancestor.define}`,
    `@override ${ancestor.override}`,
    ancestor.variables.length > 0 && `@variables ${ancestor.variables.join(', ')}`,
  ].filter(Boolean).join('\n') || undefined
}
