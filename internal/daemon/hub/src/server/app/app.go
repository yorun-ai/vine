package app

import (
	"sync"

	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/lockserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	adminapi "go.yorun.ai/vine/internal/daemon/hub/src/server/mod/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/controlapi"
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

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`

	// The schema repository is app state: every injector of this application
	// shares the same store, while separate applications stay independent.
	schemaRepoOnce sync.Once
	schemaRepo     *schema.SchemaRepo
}

func (a *HubApp) schemaRepository() *schema.SchemaRepo {
	a.schemaRepoOnce.Do(func() {
		a.schemaRepo = new(schema.SchemaRepo)
	})
	return a.schemaRepo
}

func (a *HubApp) Name() string {
	return daemon.HubIdentity.String()
}

func (a *HubApp) DIInit() {
	a.Flag.Normalize(a.InprocFlag.Enabled)
	identity := mtls.MustLoad(daemon.HubIdentity.SPIFFEPath(), a.Flag.MTLS)

	appInfo := meta.MustNewAppWithRandomId(a.Name(), buildinfo.MustVineVersion())
	a.InternalAttrs = app.InternalAttributes{
		CurrentApp:      appInfo,
		Linker:          link.NewInternalLinker(appInfo),
		BackendIdentity: identity,
		DisableConsole:  true,
		// The Admin API runs on the admin module's own listener, so the
		// application listener has nothing left to serve.
		DisableHTTPServer: true,
		ProtectHTTPServer: true,
		HTTPServerClients: []mtls.SPIFFEPath{daemon.PortalIdentity.SPIFFEPath()},
	}

	a.AppFlag.ListenAddr = a.Flag.AdminListen
}

func (a *HubApp) InitComponents(addComponent app.TypeAdder) {
	addComponent(app.T[*configaccess.Access]())
	addComponent(app.T[*repodb.HubDatabase]())
	addComponent(app.T[*natsserver.NATSServer]())
	addComponent(app.T[*lockserver.Server]())
	addComponent(app.T[*watchserver.Server]())
}

func (a *HubApp) InitModules(addModule app.TypeAdder) {
	addModule(app.T[*syncer.Syncer]())
	addModule(app.T[*seeder.Seeder]())
	addModule(app.T[*initializer.Initializer]())
	addModule(app.T[*scheduler.Scheduler]())
	addModule(app.T[*sweeper.Sweeper]())
	// Keep Control API last so reverse lifecycle shutdown stops accepting Link
	// and Portal requests before the rest of Hub begins to tear down.
	addModule(app.T[*controlapi.Server]())
	addModule(app.T[*adminapi.Server]())
}

func (a *HubApp) BindCommon(b *di.Binder) {
	b.Bind(di.T[core.MessageQueueRepo]()).ToImplementation(di.T[*repo.MessageQueueRepo]())
	b.Bind(di.T[core.AppConfigRepo]()).ToImplementation(di.T[*repo.AppConfigRepo]())
	b.Bind(di.T[core.PortalCertRepo]()).ToImplementation(di.T[*repo.PortalCertRepo]())
	b.Bind(di.T[core.PortalEntryRepo]()).ToImplementation(di.T[*repo.PortalEntryRepo]())
	b.Bind(di.T[core.PortalRuleRepo]()).ToImplementation(di.T[*repo.PortalRuleRepo]())
	b.Bind(di.T[core.PortalSiteRepo]()).ToImplementation(di.T[*repo.PortalSiteRepo]())
	b.Bind(di.T[core.MetadataRepo]()).ToImplementation(di.T[*repo.MetadataRepo]())

	b.Bind(di.T[core.SchemaRepo]()).ToInstance(a.schemaRepository())
	b.Bind(di.T[core.RegistryRepo]()).ToImplementation(di.T[*repo.RegistryRepo]())
	b.Bind(di.T[core.PortalInstanceRepo]()).ToImplementation(di.T[*repo.PortalInstanceRepo]())
}
