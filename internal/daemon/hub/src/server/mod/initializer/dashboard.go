package initializer

import (
	"fmt"
	"net"
	"strconv"

	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/util/vnet"
)

func (i *Initializer) initDashboard() {
	i.Syncer.WriteSchemas(i.SchemaRepo.ListVineHubSchemaViews())

	rpcEndpoint := i.dashboardRpcEndpoint()
	for _, serviceName := range seeder.DashboardRpcServices {
		reg := dashboardRpcRegistrations[serviceName]
		reg.Endpoint = rpcEndpoint
		i.Syncer.SyncRpcServiceRegistration(reg)
	}

	webReg := dashboardWebRegistration
	webReg.Endpoint = i.dashboardWebEndpoint()
	i.Syncer.SyncWebRegistration(seeder.DashboardWebCoreEntry.WebName, webReg)
}

func (i *Initializer) dashboardRpcEndpoint() string {
	if i.InprocFlag.Enabled {
		return rpcinproc.Endpoint(dashboardInprocPath, dashboardRpcHttpPath)
	}
	return i.dashboardHttpEndpoint(dashboardRpcHttpPath)
}

func (i *Initializer) dashboardWebEndpoint() string {
	if i.InprocFlag.Enabled {
		return webinproc.Endpoint(dashboardInprocPath, dashboardWebHttpPath)
	}
	return i.dashboardHttpEndpoint(dashboardWebHttpPath)
}

func (i *Initializer) dashboardHttpEndpoint(path string) string {
	host := vnet.MustParseHost(i.Flag.AdminListen)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = vnet.DetectHostIP()
	}
	scheme := "http"
	if i.Identity.Enabled() {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(host, strconv.Itoa(i.Flag.AdminPort())), path)
}

// Default dashboard configurations

var (
	dashboardAppName       = "vine.hub.dashboard"
	dashboardAppVersion    = "0.0.0"
	dashboardAppInstanceId = "00000000-0000-0000-0000-000000000001"

	dashboardRpcHttpPath = coreapp.PathRpcInvoke
	dashboardWebHttpPath = coreapp.PathWebAccess + "/vine.hub.admin.DashboardWeb"
	dashboardInprocPath  = hubapp.HubAdminInprocHostPath

	dashboardRpcRegistrations = newDashboardRpcRegistrations()
	dashboardWebRegistration  = redised.WebRegistration{
		ServerIdentity: mtls.HubIdentity,
		WebSkelName:    seeder.DashboardWebCoreEntry.WebName,
		AppName:        dashboardAppName,
		AppVersion:     dashboardAppVersion,
		AppInstanceId:  dashboardAppInstanceId,
	}
)

func newDashboardRpcRegistrations() map[string]redised.RpcServiceRegistration {
	registrations := make(map[string]redised.RpcServiceRegistration, len(seeder.DashboardRpcServices))
	for _, serviceName := range seeder.DashboardRpcServices {
		registrations[serviceName] = redised.RpcServiceRegistration{
			ServerIdentity: mtls.HubIdentity,
			ServiceName:    serviceName,
			AppName:        dashboardAppName,
			AppVersion:     dashboardAppVersion,
			AppInstanceId:  dashboardAppInstanceId,
		}
	}
	return registrations
}
