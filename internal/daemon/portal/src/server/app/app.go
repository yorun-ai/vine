package app

import (
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/entry"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/heartbeat"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
)

func init() {
	logger.ConfigureStdLogProcessors(
		"portal-tls-eof",
		logger.StdLogRegexpFilterProcessor(`^http: TLS handshake error from .*: EOF$`),
	)
}

type PortalApp struct {
	app.InternalApplication

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`
}

func (*PortalApp) Name() string {
	return daemon.PortalIdentity.String()
}

func (a *PortalApp) DIInit() {
	a.Flag.Normalize()
	identity := mtls.MustLoad(daemon.PortalIdentity.SPIFFEPath(), a.Flag.MTLS)

	appInfo := meta.MustNewAppWithRandomId(a.Name(), buildinfo.MustVineVersion())
	a.InternalAttrs = app.InternalAttributes{
		CurrentApp:        appInfo,
		Linker:            link.NewRedirectedInternalLinker(appInfo, a.Flag.HubEndpoint),
		BackendIdentity:   identity,
		RPCTransport:      identity.HTTPTransport(daemon.HubIdentity.SPIFFEPath()),
		DisableConsole:    true,
		DisableHTTPServer: true,
	}
}

func (*PortalApp) InitComponents(addComponent app.TypeAdder) {
	addComponent(app.T[*hubinfo.HubInfo]())
	addComponent(app.T[*hubwatch.Client]())
}

func (*PortalApp) InitModules(addModule app.TypeAdder) {
	addModule(app.T[*heartbeat.Heartbeat]())
	addModule(app.T[*epmgr.Manager]())
	addModule(app.T[*access.Access]())
	addModule(app.T[*site.Manager]())
	addModule(app.T[*vault.Vault]())
	addModule(app.T[*entry.Manager]())
}
