import type { ServiceDebugPortalInstance } from '@/skeled/admin'

export function filterPortalInstances(
  instances: Array<ServiceDebugPortalInstance>,
  query: string,
): Array<ServiceDebugPortalInstance> {
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
