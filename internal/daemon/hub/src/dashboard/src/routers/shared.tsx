import { Navigate, Outlet, useRouterState } from '@tanstack/react-router'

export function EmptyComponent() {
  return null
}

export function OutletOrRedirect({ path, to }: { path: string; to: string }) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname !== path) {
    return <Outlet />
  }

  return <Navigate to={to} replace />
}
