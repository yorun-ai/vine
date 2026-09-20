import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'
import type * as React from 'react'

import { SkeletonPage } from '@/features/skeleton/page'
import { SkeletonDomainPage } from '@/features/skeleton/skeleton-domain-page'

import { EmptyComponent } from './shared'

export function createSkeletonRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const SkeletonDomainRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/skeleton/domain',
    component: SkeletonDomainPage,
  })

  const SkeletonDomainNameRoute = createRoute({
    getParentRoute: () => SkeletonDomainRoute,
    path: '$domain',
    component: EmptyComponent,
  })

  const SkeletonDomainSchemaRoute = createRoute({
    getParentRoute: () => SkeletonDomainNameRoute,
    path: '$schemaHash',
    component: EmptyComponent,
  })

  return [
    SkeletonDomainRoute.addChildren([
      SkeletonDomainNameRoute.addChildren([SkeletonDomainSchemaRoute]),
    ]),
    createSkeletonRoute(parentRoute, 'actor', 'actors'),
    createSkeletonRoute(parentRoute, 'config', 'configs'),
    createSkeletonRoute(parentRoute, 'data', 'data'),
    createSkeletonRoute(parentRoute, 'event', 'events'),
    createSkeletonRoute(parentRoute, 'resource', 'resources'),
    createSkeletonRoute(parentRoute, 'service', 'services'),
    createSkeletonRoute(parentRoute, 'task', 'tasks'),
    createSkeletonRoute(parentRoute, 'web', 'webs'),
  ] as const
}

function createSkeletonRoute<
  TParent extends AnyRoute,
  const TPath extends string,
  const TKind extends React.ComponentProps<typeof SkeletonPage>['kind'],
>(parentRoute: TParent, path: TPath, kind: TKind) {
  const route = createRoute({
    getParentRoute: () => parentRoute,
    path: `/skeleton/${path}` as const,
    component: () => <SkeletonPage kind={kind} />,
  })
  const nameRoute = createRoute({
    getParentRoute: () => route,
    path: '$skelName',
    component: EmptyComponent,
  })
  const schemaRoute = createRoute({
    getParentRoute: () => nameRoute,
    path: '$schemaHash',
    component: EmptyComponent,
  })

  return route.addChildren([nameRoute.addChildren([schemaRoute])])
}
