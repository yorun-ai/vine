// A portal site whose Web declares a mount path is served there, so the Hub
// rejects an entry rule that targets that site unless the rule matches and
// forwards exactly that path. The Dashboard keeps both prefixes read-only and in
// sync with it instead of letting the Hub reject the rule. A Web without a mount
// path does not limit the rules of its site.

export interface WebMountPathSite {
  name: string
  type: string
  webMountPath: string
}

// lockedWebMountPath returns the mount path a rule must use, or null when the
// rule is free to choose its own prefixes because its site is not limited.
export function lockedWebMountPath(
  routeType: string,
  routeSiteName: string,
  sites: Array<WebMountPathSite>,
): string | null {
  if (routeType !== 'SITE') {
    return null
  }

  const site = sites.find((entry) => entry.name === routeSiteName)
  if (!site || site.type !== 'WEBGW') {
    return null
  }

  return site.webMountPath.trim() || null
}

export function lockWebMountPath<
  T extends { matchPathPrefix: string; routePathPrefix: string },
>(value: T, mountPath: string | null): T {
  if (
    mountPath === null ||
    (value.matchPathPrefix === mountPath &&
      value.routePathPrefix === mountPath)
  ) {
    return value
  }

  return { ...value, matchPathPrefix: mountPath, routePathPrefix: mountPath }
}
