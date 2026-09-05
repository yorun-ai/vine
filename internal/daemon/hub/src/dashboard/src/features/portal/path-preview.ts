export type PathPreview =
  | { kind: 'empty' | 'invalidRequest' | 'invalidRoute' | 'noMatch' }
  | { kind: 'matched'; path: string }

export function previewRulePath(
  request: string,
  matchPathPrefix: string,
  routePathPrefix: string,
): PathPreview {
  if (!request) return { kind: 'empty' }
  // Keep the query verbatim; URL normalization would change dot segments and
  // escaped separators before the preview can apply Portal's matching rules.
  const queryAt = request.indexOf('?')
  const path = queryAt < 0 ? request : request.slice(0, queryAt)
  const query = queryAt < 0 ? '' : request.slice(queryAt)
  if (!path.startsWith('/') || /[#\s\x00-\x1f\x7f]/.test(request)) {
    return { kind: 'invalidRequest' }
  }
  let decoded: string
  try {
    decoded = decodeURIComponent(path)
  } catch {
    return { kind: 'invalidRequest' }
  }
  try {
    const route = decodeURIComponent(routePathPrefix)
    if (routePathPrefix && (
      !routePathPrefix.startsWith('/') || routePathPrefix.startsWith('//') ||
      /[?#\s]/.test(routePathPrefix) || /[\\\x00-\x1f\x7f]/.test(route) ||
      route.split('/').some((part) => part === '.' || part === '..')
    )) return { kind: 'invalidRoute' }
  } catch {
    return { kind: 'invalidRoute' }
  }
  const rootMatch = matchPathPrefix === '' || matchPathPrefix === '/'
  if (!rootMatch && decoded !== matchPathPrefix && !decoded.startsWith(`${matchPathPrefix}/`)) {
    return { kind: 'noMatch' }
  }
  // Consume the matched decoded characters while preserving the escaped suffix.
  let offset = 0
  if (!rootMatch) {
    let consumed = ''
    while (consumed.length < matchPathPrefix.length) {
      const start = offset
      if (path[offset] === '%') {
        // Accumulate a complete UTF-8 character, possibly spanning several escapes.
        do {
          offset += 3
          try {
            consumed += decodeURIComponent(path.slice(start, offset))
            break
          } catch {
            // The complete path was validated above, so another escape completes it.
          }
        } while (offset < path.length)
      } else {
        consumed += path[offset++]
      }
    }
  }
  const target = routePathPrefix.replace(/\/+$/, '') + path.slice(offset)
  return { kind: 'matched', path: (target || '/') + query }
}
