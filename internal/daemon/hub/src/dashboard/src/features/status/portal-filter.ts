import type { PortalStatusView } from '@/skeled/admin'

export function filterPortalInstances(
  instances: Array<PortalStatusView>,
  query: string,
): Array<PortalStatusView> {
  const keyword = query.trim().toLowerCase()
  if (keyword === '') {
    return instances
  }

  return instances.filter(
    (instance) =>
      instance.instanceId.toLowerCase().includes(keyword) ||
      instance.version.toLowerCase().includes(keyword),
  )
}
