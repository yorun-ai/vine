export function isConfigDateTime(type: string) {
  return /^(localdate|localtime|localdatetime|timestamp)\??$/.test(type)
}

export interface ConfigDateTimeParts {
  date: string
  time: string
  fraction: string
  offset: string
}

export function splitConfigDateTime(type: string, value: unknown): ConfigDateTimeParts {
  const text = typeof value === 'string' ? value : ''
  const base = type.replace(/\?$/, '')
  const match = /^(?:(\d{4}-\d{2}-\d{2})T)?(\d{2}:\d{2}(?::\d{2})?)(?:\.(\d{1,9}))?(Z|[+-]\d{2}:\d{2})?$/.exec(text)
  return {
    date: base === 'localdate' ? text : match?.[1] ?? '',
    time: match?.[2] ?? '',
    fraction: match?.[3] ?? '',
    offset: match?.[4] ?? 'Z',
  }
}

export function formatConfigDateTime(type: string, parts: ConfigDateTimeParts) {
  const base = type.replace(/\?$/, '')
  if (base !== 'localtime') {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(parts.date)) {
      return null
    }
    const date = new Date(`${parts.date}T00:00:00Z`)
    if (!Number.isFinite(date.getTime()) || date.toISOString().slice(0, 10) !== parts.date) {
      return null
    }
  }
  if (base === 'localdate') {
    return parts.date
  }
  if (!/^(?:[01]\d|2[0-3]):[0-5]\d(?::[0-5]\d)?$/.test(parts.time) || !/^\d{0,9}$/.test(parts.fraction)) {
    return null
  }
  const time = `${parts.time.length === 5 ? `${parts.time}:00` : parts.time}${parts.fraction ? `.${parts.fraction}` : ''}`
  if (base === 'localtime') {
    return time
  }
  if (base === 'timestamp' && !/^(Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$/.test(parts.offset)) {
    return null
  }
  return `${parts.date}T${time}${base === 'timestamp' ? parts.offset : ''}`
}
