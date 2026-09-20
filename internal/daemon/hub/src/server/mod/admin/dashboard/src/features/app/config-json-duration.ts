export const configDurationUnits = ['ns', 'us', 'ms', 's', 'm', 'h'] as const

export function formatConfigDuration(amount: string, unit: string) {
  if (!configDurationUnits.some((item) => item === unit) ||
    !/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)$/.test(amount) ||
    !Number.isFinite(Number(amount))) {
    return null
  }
  return `${amount}${unit}`
}

export function splitConfigDuration(value: unknown) {
  const match = typeof value === 'string'
    ? /^([+-]?(?:\d+(?:\.\d*)?|\.\d+))(ns|us|µs|μs|ms|s|m|h)$/.exec(value)
    : null
  return match
    ? { amount: match[1], unit: match[2] === 'µs' || match[2] === 'μs' ? 'us' : match[2] }
    : { amount: '', unit: 's' }
}
