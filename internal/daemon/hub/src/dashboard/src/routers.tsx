import {
  Navigate,
  Outlet,
  createRootRoute,
  createRoute,
} from '@tanstack/react-router'

import './styles.css'
import { Toaster } from '@/components/ui/sonner'
import { LocaleProvider } from '@/i18n'
import { TanstackQueryRootProvider } from '@/lib/query-client'
import { AppLayout } from '@/layouts/app-layout/app-layout'

import { createAppRoutes } from './routers/app'
import { createDebugRoutes } from './routers/debug'
import { createMaintenanceRoutes } from './routers/maintenance'
import { createPortalRoutes } from './routers/portal'
import { createSettingsRoutes } from './routers/settings'
import { createSkeletonRoutes } from './routers/skeleton'
import { createStatusRoutes } from './routers/status'

export const RootRoute = createRootRoute({
  component: RootComponent,
})

const AuthenticatedRoute = createRoute({
  getParentRoute: () => RootRoute,
  id: '_authenticated',
  component: AuthenticatedComponent,
})

const HomeRoute = createRoute({
  getParentRoute: () => AuthenticatedRoute,
  path: '/',
  component: HomeComponent,
})

export const routeTree = RootRoute.addChildren([
  AuthenticatedRoute.addChildren([
    HomeRoute,
    createAppRoutes(AuthenticatedRoute),
    createStatusRoutes(AuthenticatedRoute),
    createDebugRoutes(AuthenticatedRoute),
    createPortalRoutes(AuthenticatedRoute),
    createMaintenanceRoutes(AuthenticatedRoute),
    createSettingsRoutes(AuthenticatedRoute),
    ...createSkeletonRoutes(AuthenticatedRoute),
  ]),
])

function RootComponent() {
  return (
    <LocaleProvider>
      <TanstackQueryRootProvider>
        <Outlet />
        <Toaster position="top-right" closeButton />
      </TanstackQueryRootProvider>
    </LocaleProvider>
  )
}

function AuthenticatedComponent() {
  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  )
}

function HomeComponent() {
  return <Navigate to="/app/config" replace />
}
