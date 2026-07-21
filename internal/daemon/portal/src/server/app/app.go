package app

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/entry"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
)

type PortalApp struct {
	app.InternalApplication

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`
}

func (*PortalApp) Name() string {
	return "vine.portal"
}

func (a *PortalApp) DIInit() {
	a.Flag.Normalize()

	appName := a.Name()
	if a.InprocFlag.Enabled {
		appName += "@" + runtime.Application().Name()
	}
	appInfo := meta.MustNewAppWithRandomId(appName, runtime.Application().Version())
	a.InternalAttrs = app.InternalAttributes{
		Info:              appInfo,
		Linker:            link.NewRedirectedInternalLinker(appInfo, a.Flag.HubEndpoint),
		DisableConsole:    true,
		DisableHTTPServer: true,
	}
}

func (*PortalApp) InitComponents(addComponent app.TypeAdder) {
	addComponent(app.T[*hubinfo.HubInfo]())
	addComponent(app.T[*hubredis.Client]())
}

func (*PortalApp) InitModules(addModule app.TypeAdder) {
	addModule(app.T[*epmgr.Manager]())
	addModule(app.T[*access.Access]())
	addModule(app.T[*site.Manager]())
	addModule(app.T[*vault.Vault]())
	addModule(app.T[*entry.Manager]())
}
