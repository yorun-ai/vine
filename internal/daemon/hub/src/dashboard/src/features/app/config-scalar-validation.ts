const durationUnits: Record<string, bigint> = {
  ns: 1n, us: 1000n, 'µs': 1000n, 'μs': 1000n,
  ms: 1000000n, s: 1000000000n, m: 60000000000n, h: 3600000000000n,
}

function validDuration(text: string) {
  const negative = text.startsWith('-')
  let rest = text.replace(/^[+-]/, '')
  if (rest === '0') {
    return true
  }
  if (!rest) {
    return false
  }
  let total = 0n
  const limit = negative ? 1n << 63n : (1n << 63n) - 1n
  while (rest) {
    const match = /^(\d+(?:\.\d*)?|\.\d+)(ns|us|µs|μs|ms|s|m|h)/.exec(rest)
    if (!match) {
      return false
    }
    const [integer, fraction = ''] = match[1].split('.')
    const unit = durationUnits[match[2]]
    total += BigInt(integer || '0') * unit
    if (fraction) {
      total += BigInt(fraction) * unit / (10n ** BigInt(fraction.length))
    }
    if (total > limit) {
      return false
    }
    rest = rest.slice(match[0].length)
  }
  return true
}

function validDate(text: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) {
    return false
  }
  const date = new Date(`${text}T00:00:00Z`)
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === text
}

function validTime(text: string) {
  return /^(?:[01]?\d|2[0-3]):[0-5]\d:[0-5]\d(?:[.,]\d+)?$/.test(text)
}

export function validConfigUUID(text: string) {
  const canonical = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
  return new RegExp(`^(?:${canonical}|\\{${canonical}\\}|urn:uuid:${canonical}|[0-9a-f]{32})$`, 'i').test(text)
}

export function configScalarFormatMatches(value: unknown, type: string) {
  if (typeof value !== 'string') {
    return false
  }
  switch (type) {
    case 'duration':
      return validDuration(value)
    case 'uuid':
      return validConfigUUID(value)
    case 'decimal': {
      const match = /^[+-]?(?:\d+(?:\.(\d*))?|\.(\d+))(?:[eE]([+-]?\d+))?$/.exec(value)
      if (!match) {
        return false
      }
      const exponent = BigInt(match[3] ?? '0')
      const scale = exponent - BigInt((match[1] ?? match[2] ?? '').length)
      return exponent >= -2147483648n && exponent <= 2147483647n && scale >= -2147483648n && scale <= 2147483647n
    }
    case 'localdate':
      return validDate(value)
    case 'localtime':
      return validTime(value)
    case 'localdatetime': {
      const parts = value.split(/[Tt]/)
      return parts.length === 2 && validDate(parts[0]) && validTime(parts[1])
    }
    case 'timestamp': {
      const match = /^(\d{4}-\d{2}-\d{2})T(.+?)(Z|[+-]\d{2}:\d{2})$/.exec(value)
      if (!match || !validDate(match[1]) || !validTime(match[2])) {
        return false
      }
      return match[3] === 'Z' || (Number(match[3].slice(1, 3)) <= 24 && Number(match[3].slice(4)) <= 60)
    }
    default:
      return true
  }
}
