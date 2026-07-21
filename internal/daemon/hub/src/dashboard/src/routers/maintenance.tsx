import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import {
  DataUpdatePreviewPage,
  DataUpdateResultPage,
  DataUpdateUploadPage,
} from '@/features/maintenance/data-update-flow'

import { OutletOrRedirect } from './shared'

export function createMaintenanceRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const MaintenanceRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/maintenance',
    component: MaintenanceComponent,
  })

  const DataUpdateRoute = createRoute({
    getParentRoute: () => MaintenanceRoute,
    path: 'data-update',
    component: DataUpdateComponent,
  })

  const DataUpdateUploadRoute = createRoute({
    getParentRoute: () => DataUpdateRoute,
    path: 'upload',
    component: DataUpdateUploadPage,
  })

  const DataUpdatePreviewRoute = createRoute({
    getParentRoute: () => DataUpdateRoute,
    path: 'preview',
    component: DataUpdatePreviewPage,
  })

  const DataUpdateResultRoute = createRoute({
    getParentRoute: () => DataUpdateRoute,
    path: 'result',
    component: DataUpdateResultPage,
  })

  return MaintenanceRoute.addChildren([
    DataUpdateRoute.addChildren([
      DataUpdateUploadRoute,
      DataUpdatePreviewRoute,
      DataUpdateResultRoute,
    ]),
  ])
}

function MaintenanceComponent() {
  return <OutletOrRedirect path="/maintenance" to="/maintenance/data-update" />
}

function DataUpdateComponent() {
  return (
    <OutletOrRedirect
      path="/maintenance/data-update"
      to="/maintenance/data-update/upload"
    />
  )
}
