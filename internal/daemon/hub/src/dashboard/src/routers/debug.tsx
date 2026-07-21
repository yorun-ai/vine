import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { EventEmitterPage } from '@/features/debug/event-emitter-page'
import { ServiceClientPage } from '@/features/debug/service-client-page'
import { TaskLauncherPage } from '@/features/debug/task-launcher-page'

import { OutletOrRedirect } from './shared'

export function createDebugRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const DebugRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/debug',
    component: DebugComponent,
  })

  const ServiceClientRoute = createRoute({
    getParentRoute: () => DebugRoute,
    path: 'service-client',
    component: ServiceClientPage,
  })

  const ServiceClientServiceRoute = createRoute({
    getParentRoute: () => ServiceClientRoute,
    path: '$serviceSkelName',
    component: ServiceClientPage,
  })

  const ServiceClientMethodRoute = createRoute({
    getParentRoute: () => ServiceClientServiceRoute,
    path: '$methodSkelName',
    component: ServiceClientPage,
  })

  const TaskLauncherRoute = createRoute({
    getParentRoute: () => DebugRoute,
    path: 'task-launcher',
    component: TaskLauncherPage,
  })

  const TaskLauncherTaskRoute = createRoute({
    getParentRoute: () => TaskLauncherRoute,
    path: '$taskSkelName',
    component: TaskLauncherPage,
  })

  const TaskLauncherTriggerRoute = createRoute({
    getParentRoute: () => TaskLauncherTaskRoute,
    path: '$triggerSkelName',
    component: TaskLauncherPage,
  })

  const EventEmitterRoute = createRoute({
    getParentRoute: () => DebugRoute,
    path: 'event-emitter',
    component: EventEmitterPage,
  })

  const EventEmitterEventRoute = createRoute({
    getParentRoute: () => EventEmitterRoute,
    path: '$eventSkelName',
    component: EventEmitterPage,
  })

  return DebugRoute.addChildren([
    ServiceClientRoute.addChildren([
      ServiceClientServiceRoute.addChildren([ServiceClientMethodRoute]),
    ]),
    TaskLauncherRoute.addChildren([
      TaskLauncherTaskRoute.addChildren([TaskLauncherTriggerRoute]),
    ]),
    EventEmitterRoute.addChildren([EventEmitterEventRoute]),
  ])
}

function DebugComponent() {
  return <OutletOrRedirect path="/debug" to="/debug/service-client" />
}
