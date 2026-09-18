import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { MessageQueuePage } from '@/features/status/message-queue-page'
import { AppStatusPage } from '@/features/status/app-page'
import { PortalInstancePage } from '@/features/status/portal-page'

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

  const StatusPortalRoute = createRoute({
    getParentRoute: () => StatusRoute,
    path: 'portal',
    component: PortalInstancePage,
  })

  const StatusPortalInstanceRoute = createRoute({
    getParentRoute: () => StatusPortalRoute,
    path: '$instanceId',
    component: PortalInstancePage,
  })

  const MessageQueueRoute = createRoute({
    getParentRoute: () => StatusRoute,
    path: 'message-queue',
    component: MessageQueuePage,
  })

  return StatusRoute.addChildren([
    MessageQueueRoute,
    StatusAppRoute.addChildren([StatusAppInstanceRoute]),
    StatusPortalRoute.addChildren([StatusPortalInstanceRoute]),
  ])
}

function StatusComponent() {
  return <OutletOrRedirect path="/status" to="/status/app" />
}
