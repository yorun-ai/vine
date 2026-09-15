// Web mount paths determine the effective match and route prefixes. The
// Dashboard displays both the configured and effective prefixes; configured
// prefixes are only used by sites whose Web has no mount path.

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

export function effectiveWebMountPrefixes(mountPath: string): {
  matchPathPrefix: string
  routePathPrefix: string
} {
  const prefix = mountPath.replace(/\/+$/, '')
  return {
    matchPathPrefix: prefix || '/',
    routePathPrefix: prefix,
  }
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

// Do not persist the displayed Web path as a second configuration source.
export function rulePathsForSave<
  T extends { matchPathPrefix: string; routePathPrefix: string },
>(value: T, _mountPath: string | null): T {
  return value
}
