import type * as React from 'react'
import {
  Braces,
  CalendarClock,
  Globe2,
  Radio,
  Server,
  ShieldCheck,
  SlidersHorizontal,
  Users,
} from 'lucide-react'

import { vrpcClient } from '@/config/vrpc-client'
import { createSkeletonService } from '@/skeled'
import type {
  SkeletonActorItem,
  SkeletonConfigItem,
  SkeletonData,
  SkeletonEventItem,
  SkeletonResourceItem,
  SkeletonServiceItem,
  SkeletonTask,
  SkeletonWebItem,
} from '@/skeled'

export const skeletonService = createSkeletonService(vrpcClient)

export type SkeletonKind =
  | 'actors'
  | 'configs'
  | 'services'
  | 'resources'
  | 'data'
  | 'webs'
  | 'tasks'
  | 'events'

export type SkeletonItem =
  | SkeletonActorItem
  | SkeletonConfigItem
  | SkeletonServiceItem
  | SkeletonResourceItem
  | SkeletonData
  | SkeletonWebItem
  | SkeletonTask
  | SkeletonEventItem

export type TypeDefinitionIndex = Map<string, Array<SkeletonData>>

export const skeletonConfigs: Record<
  SkeletonKind,
  {
    title: string
    description: string
    emptyTitle: string
    icon: React.ComponentType<{ className?: string }>
    load: () => Promise<Array<SkeletonItem>>
  }
> = {
  actors: {
    title: 'Actor',
    description: 'View declared actors and their supported vias.',
    emptyTitle: 'No Actor',
    icon: Users,
    load: () => skeletonService.listActors(null),
  },
  configs: {
    title: 'Config',
    description: 'View config lifecycles, fields, and versioned definitions.',
    emptyTitle: 'No Config',
    icon: SlidersHorizontal,
    load: () => skeletonService.listConfigs(null),
  },
  services: {
    title: 'Service',
    description: 'View RPC services, accessible actors, visibility, and methods.',
    emptyTitle: 'No Service',
    icon: Server,
    load: () => skeletonService.listServices(null),
  },
  resources: {
    title: 'Resource',
    description: 'View resources, permission codes, checks, and expressions.',
    emptyTitle: 'No Resource',
    icon: ShieldCheck,
    load: () => skeletonService.listResources(null),
  },
  data: {
    title: 'Data',
    description: 'View Data and Enum fields, type parameters, and enum items.',
    emptyTitle: 'No Data',
    icon: Braces,
    load: () => skeletonService.listData(null),
  },
  webs: {
    title: 'Web',
    description: 'View Web sites and their accessible actors.',
    emptyTitle: 'No Web',
    icon: Globe2,
    load: () => skeletonService.listWebs(null),
  },
  tasks: {
    title: 'Task',
    description: 'View background tasks, triggers, and input parameters.',
    emptyTitle: 'No Task',
    icon: CalendarClock,
    load: () => skeletonService.listTasks(null),
  },
  events: {
    title: 'Event',
    description: 'View event definitions, fields, and publication scope.',
    emptyTitle: 'No Event',
    icon: Radio,
    load: () => skeletonService.listEvents(null),
  },
}

export function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

export function itemValues(item: SkeletonItem) {
  const values = [item.skelName, item.schemaHash]
  if ('actions' in item) {
    values.push(...item.actions.map((action) => action.permissionCode))
  }
  return values
}

export function itemVersionKey(item: SkeletonItem) {
  return !item.isMain ? `${item.skelName}:${item.schemaHash}` : item.skelName
}

export function itemRouteHash(item: SkeletonItem) {
  return !item.isMain ? item.schemaHash : undefined
}

function encodePathSegment(value: string) {
  return encodeURIComponent(value)
}

export function skeletonItemHref(item: SkeletonItem, kind: SkeletonKind) {
  const routeHash = itemRouteHash(item)
  const basePath = `${skeletonRouteConfig[kind].listPath}/${encodePathSegment(item.skelName)}`
  return routeHash ? `${basePath}/${encodePathSegment(routeHash)}` : basePath
}

export function skeletonActorHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/actor/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonServiceHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/service/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonResourceHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/resource/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonWebHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/web/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonEventHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/event/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonTaskHref(skelName: string, schemaHash?: string) {
  const basePath = `/skeleton/task/${encodePathSegment(skelName)}`
  return schemaHash ? `${basePath}/${encodePathSegment(schemaHash)}` : basePath
}

export function skeletonDomainHref(item: SkeletonItem) {
  return `/skeleton/domain/${encodePathSegment(item.domain)}`
}

export function displayItemName(item: SkeletonItem) {
  if ('typeParameters' in item && item.typeParameters.length > 0) {
    return `${item.name}<${item.typeParameters.join(', ')}>`
  }
  return item.name
}

export function splitDomainSkelName(domain: string, skelName: string) {
  const prefix = `${domain}.`
  if (!skelName.startsWith(prefix)) {
    return { domainPart: domain, restPart: skelName.slice(domain.length) }
  }
  return { domainPart: domain, restPart: skelName.slice(domain.length) }
}

export function buildTypeDefinitionIndex(items: Array<SkeletonData>) {
  const index: TypeDefinitionIndex = new Map()
  for (const item of items) {
    addTypeDefinition(index, item.skelName, item)
  }
  return index
}

function addTypeDefinition(
  index: TypeDefinitionIndex,
  key: string,
  item: SkeletonData,
) {
  const values = index.get(key) ?? []
  values.push(item)
  index.set(key, values)
}

export function findTypeDefinition(
  index: TypeDefinitionIndex,
  key: string,
  domainSchemaHash?: string,
): SkeletonData | null {
  const values = index.get(key) ?? []
  const sameDomainVersion =
    domainSchemaHash === undefined
      ? undefined
      : values.find((item) => item.domainSchemaHash === domainSchemaHash)
  return (
    sameDomainVersion ??
    values.find((item) => item.isMain) ??
    (values.length > 0 ? values[0] : undefined) ??
    null
  )
}

export const skeletonRouteConfig: Record<
  SkeletonKind,
  { listPath: string; detailPath: string; detailVersionPath: string }
> = {
  actors: {
    listPath: '/skeleton/actor',
    detailPath: '/skeleton/actor/$skelName',
    detailVersionPath: '/skeleton/actor/$skelName/$schemaHash',
  },
  configs: {
    listPath: '/skeleton/config',
    detailPath: '/skeleton/config/$skelName',
    detailVersionPath: '/skeleton/config/$skelName/$schemaHash',
  },
  services: {
    listPath: '/skeleton/service',
    detailPath: '/skeleton/service/$skelName',
    detailVersionPath: '/skeleton/service/$skelName/$schemaHash',
  },
  resources: {
    listPath: '/skeleton/resource',
    detailPath: '/skeleton/resource/$skelName',
    detailVersionPath: '/skeleton/resource/$skelName/$schemaHash',
  },
  data: {
    listPath: '/skeleton/data',
    detailPath: '/skeleton/data/$skelName',
    detailVersionPath: '/skeleton/data/$skelName/$schemaHash',
  },
  webs: {
    listPath: '/skeleton/web',
    detailPath: '/skeleton/web/$skelName',
    detailVersionPath: '/skeleton/web/$skelName/$schemaHash',
  },
  tasks: {
    listPath: '/skeleton/task',
    detailPath: '/skeleton/task/$skelName',
    detailVersionPath: '/skeleton/task/$skelName/$schemaHash',
  },
  events: {
    listPath: '/skeleton/event',
    detailPath: '/skeleton/event/$skelName',
    detailVersionPath: '/skeleton/event/$skelName/$schemaHash',
  },
}
