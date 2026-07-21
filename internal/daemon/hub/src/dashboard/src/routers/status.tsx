import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { AppStatusPage } from '@/features/status/app-page'

import { OutletOrRedirect } from './shared'

export function createStatusRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const StatusRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/status',
    component: StatusComponent,
  })

  const StatusAppRoute = createRoute({
    getParentRoute: () => StatusRoute,
    path: 'app',
    component: AppStatusPage,
  })

  const StatusAppInstanceRoute = createRoute({
    getParentRoute: () => StatusAppRoute,
    path: '$instanceId',
    component: AppStatusPage,
  })

  return StatusRoute.addChildren([
    StatusAppRoute.addChildren([StatusAppInstanceRoute]),
  ])
}

function StatusComponent() {
  return <OutletOrRedirect path="/status" to="/status/app" />
}
