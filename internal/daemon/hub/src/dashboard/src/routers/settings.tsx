import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { DashboardSettingsPage } from '@/features/maintenance/dashboard-page'

import { OutletOrRedirect } from './shared'

export function createSettingsRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const SettingsRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/settings',
    component: SettingsComponent,
  })

  const DashboardPortRoute = createRoute({
    getParentRoute: () => SettingsRoute,
    path: 'dashboard-port',
    component: DashboardSettingsPage,
  })

  return SettingsRoute.addChildren([DashboardPortRoute])
}

function SettingsComponent() {
  return <OutletOrRedirect path="/settings" to="/settings/dashboard-port" />
}
