import { createRoute } from '@tanstack/react-router'
import type { AnyRoute } from '@tanstack/react-router'

import { PortalCertPage } from '@/features/portal/cert-page'
import { PortalEntryPage } from '@/features/portal/entry-page'
import { PortalRulePage } from '@/features/portal/rule-page'
import { PortalSitePage } from '@/features/portal/site-page'

import { OutletOrRedirect } from './shared'

export function createPortalRoutes<TParent extends AnyRoute>(
  parentRoute: TParent,
) {
  const PortalRoute = createRoute({
    getParentRoute: () => parentRoute,
    path: '/portal',
    component: PortalComponent,
  })

  const PortalEntryRoute = createRoute({
    getParentRoute: () => PortalRoute,
    path: 'entry',
    component: PortalEntryPage,
  })
  const PortalEntryNameRoute = createRoute({
    getParentRoute: () => PortalEntryRoute,
    path: '$entryName',
    component: PortalEntryPage,
  })

  const PortalSiteRoute = createRoute({
    getParentRoute: () => PortalRoute,
    path: 'site',
    component: PortalSitePage,
  })
  const PortalSiteIdRoute = createRoute({
    getParentRoute: () => PortalSiteRoute,
    path: '$siteId',
    component: PortalSitePage,
  })

  const PortalRuleRoute = createRoute({
    getParentRoute: () => PortalRoute,
    path: 'rule',
    component: PortalRulePage,
  })
  const PortalRuleIdRoute = createRoute({
    getParentRoute: () => PortalRuleRoute,
    path: '$ruleId',
    component: PortalRulePage,
  })

  const PortalCertRoute = createRoute({
    getParentRoute: () => PortalRoute,
    path: 'cert',
    component: PortalCertPage,
  })
  const PortalCertIdRoute = createRoute({
    getParentRoute: () => PortalCertRoute,
    path: '$certId',
    component: PortalCertPage,
  })

  return PortalRoute.addChildren([
    PortalEntryRoute.addChildren([PortalEntryNameRoute]),
    PortalSiteRoute.addChildren([PortalSiteIdRoute]),
    PortalRuleRoute.addChildren([PortalRuleIdRoute]),
    PortalCertRoute.addChildren([PortalCertIdRoute]),
  ])
}

function PortalComponent() {
  return <OutletOrRedirect path="/portal" to="/portal/entry" />
}
