package app

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/impl"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/impl/dashboard"
	debugimpl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/debug"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/initializer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/scheduler"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/seeder"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/sweeper"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo"
	repodb "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/schema"
)

type HubApp struct {
	app.InternalApplication
	app.ServicerEnabled
	app.WebberEnabled

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`
}

func (a *HubApp) Name() string {
	return "vine.hub"
}

func (a *HubApp) DIInit() {
	a.Flag.Normalize(a.InprocFlag.Enabled)

	appName := a.Name()
	if a.InprocFlag.Enabled {
		appName += "@" + runtime.Application().Name()
	}
	appInfo := meta.MustNewAppWithRandomId(appName, runtime.Application().Version())
	a.InternalAttrs = app.InternalAttributes{
		Info:           appInfo,
		Linker:         link.NewInternalLinker(appInfo),
		DisableConsole: true,
		InprocHostPath: hubapp.HubInprocHostPath,
	}

	a.AppFlag.ListenAddr = a.Flag.APIListen
}

func (a *HubApp) InitComponents(addComponent app.TypeAdder) {
	addComponent(app.T[*repodb.HubDatabase]())
	addComponent(app.T[*natsserver.NATSServer]())
	addComponent(app.T[*redisserver.Server]())
}

func (a *HubApp) InitModules(addModule app.TypeAdder) {
	addModule(app.T[*syncer.Syncer]())
	addModule(app.T[*seeder.Seeder]())
	addModule(app.T[*initializer.Initializer]())
	addModule(app.T[*scheduler.Scheduler]())
	addModule(app.T[*sweeper.Sweeper]())
}

func (a *HubApp) BindCommon(b *di.Binder) {
	b.Bind(di.T[core.AppConfigRepo]()).ToImplementation(di.T[*repo.DBAppConfigRepo]())
	b.Bind(di.T[core.PortalCertRepo]()).ToImplementation(di.T[*repo.DBPortalCertRepo]())
	b.Bind(di.T[core.PortalRuleRepo]()).ToImplementation(di.T[*repo.DBPortalRuleRepo]())
	b.Bind(di.T[core.PortalSiteRepo]()).ToImplementation(di.T[*repo.DBPortalSiteRepo]())
	b.Bind(di.T[core.MetadataRepo]()).ToImplementation(di.T[*repo.DBMetadataRepo]())

	b.Bind(di.T[core.SchemaRepo]()).ToImplementation(di.T[*schema.MemorySchemaRepo]())
	b.Bind(di.T[core.RegistryRepo]()).ToImplementation(di.T[*repo.RedisRegistryRepo]())
}

func (*HubApp) ServicerInitHandlers(addHandler app.TypeAdder) {
	addHandler(app.T[*impl.InfoServiceServerImpl]())
	addHandler(app.T[*debugimpl.ServiceDebugServiceServerImpl]())
	addHandler(app.T[*debugimpl.TaskDebugServiceServerImpl]())
	addHandler(app.T[*debugimpl.EventDebugServiceServerImpl]())
	addHandler(app.T[*impl.SkeletonServiceServerImpl]())
	addHandler(app.T[*impl.AppStatusServiceServerImpl]())
	addHandler(app.T[*impl.AppConfigServiceServerImpl]())
	addHandler(app.T[*impl.PortalCertServiceServerImpl]())
	addHandler(app.T[*impl.PortalEntryServiceServerImpl]())
	addHandler(app.T[*impl.PortalRuleServiceServerImpl]())
	addHandler(app.T[*impl.MaintenanceServiceServerImpl]())
	addHandler(app.T[*impl.PortalSiteServiceServerImpl]())
	addHandler(app.T[*impl.RegistryServiceServerImpl]())
}

func (*HubApp) WebberInitHandlers(addHandler app.TypeAdder) {
	addHandler(app.T[*dashboard.WebServerImpl]())
}
