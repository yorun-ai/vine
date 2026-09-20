import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { AppConfigPage } from '@/features/app/config-page'

import { EmptyComponent } from './shared'

export function createAppRoutes<TParent extends AnyRoute>(parentRoute: TParent) {
  const AppConfigRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/app/config',
    component: AppConfigPage,
  })

  const AppConfigKeyRoute = createRoute({
    getParentRoute: () => AppConfigRoute,
    path: '$configKey',
    component: EmptyComponent,
  })

  return AppConfigRoute.addChildren([AppConfigKeyRoute])
}
