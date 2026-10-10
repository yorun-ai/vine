// Keep the creation-name shape aligned with the Hub's isValidConfigSkelName.
export function isValidConfigSkelName(skelName: string) {
  const parts = skelName.split('.')
  return (
    parts.length >= 2 &&
    /^.+Config(?:V[1-9][0-9]*)?$/.test(parts.at(-1) ?? '') &&
    parts.every((part) => /^[A-Za-z_]/.test(part) && !/[^A-Za-z0-9_]/.test(part))
  )
}
