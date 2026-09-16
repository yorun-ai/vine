package initializer

import (
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vslice"
)

// TODO: Delete this file once the upgrade window closes, together with the
// built-in cleanup in the model: a Watch store that never held these keys needs
// no removal, and a literal list would express the one-off pass better than the
// schema scan below.

// Hub used to publish its own Dashboard through Portal: it provisioned a
// built-in entry, two sites, and two rules, and registered the Dashboard's Rpc
// and Web endpoints in Watch. Hub serves the Admin API and the Dashboard on its
// own listener now, so the keys an earlier release published would keep routing
// the Dashboard through Portal. Hub removes them on startup; the stored entities
// go with the model's built-in cleanup, and the work is a no-op once a Watch
// store holds none.
const (
	legacyDashboardAppName       = "vine.hub.dashboard"
	legacyDashboardAppInstanceId = "00000000-0000-0000-0000-000000000001"
	legacyDashboardWebSkelName   = "vine.hub.admin.DashboardWeb"
)

var (
	legacyDashboardRuleNames = []string{"vine.hub.admin-api", "vine.hub.dashboard-web"}
	legacyDashboardSiteNames = []string{
		"vine.hub.admin.AdminActor-client-rpc",
		"vine.hub.admin.DashboardWeb-web",
	}
)

func (i *Initializer) removeLegacyDashboard() {
	for _, name := range legacyDashboardRuleNames {
		i.Syncer.WatchServer.DeleteAndNotify(watched.FormatPortalRuleKey(name))
	}
	for _, name := range legacyDashboardSiteNames {
		i.Syncer.WatchServer.DeleteAndNotify(watched.FormatPortalSiteKey(name))
	}
	for _, serviceName := range legacyDashboardRpcServiceNames() {
		i.Syncer.WatchServer.DeleteAndNotify(watched.FormatRpcServiceRegistrationKey(
			serviceName, legacyDashboardAppName, legacyDashboardAppInstanceId))
	}
	i.Syncer.WatchServer.DeleteAndNotify(watched.FormatWebRegistrationKey(
		legacyDashboardWebSkelName, legacyDashboardAppName, legacyDashboardAppInstanceId))
}

// legacyDashboardRpcServiceNames returns the Hub admin services the Dashboard
// used to register, derived the way that release derived them.
func legacyDashboardRpcServiceNames() []string {
	var names []string
	adminActorSkelName := skeled.AdminActor{}.SkelName()
	for _, domainSchema := range skel.RegisteredDomainSchemas() {
		if domainSchema.Domain != "vine.hub.admin" {
			continue
		}
		for _, service := range domainSchema.Services {
			if service.Pub {
				continue
			}
			for _, actor := range service.Audiences {
				if actor.SkelName == adminActorSkelName {
					names = append(names, service.SkelName)
					break
				}
			}
		}
	}
	return vslice.SortBy(names, func(a string, b string) bool { return a < b })
}
