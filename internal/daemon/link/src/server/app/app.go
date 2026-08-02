package app

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/impl"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/config"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/event"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/ingress"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/task"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/webproxy"
)

type LinkApp struct {
	app.InternalApplication
	app.ServicerEnabled

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`
}

func (a *LinkApp) Name() string {
	return daemon.LinkIdentity.String()
}

func (a *LinkApp) DIInit() {
	a.Flag.Normalize(a.InprocFlag.Enabled)
	identity := mtls.MustLoad(daemon.LinkIdentity.SPIFFEPath(), a.Flag.MTLS)
	a.AppFlag.ListenAddr = a.Flag.APIListen

	appInfo := meta.MustNewAppWithRandomId(a.Name(), runtime.Application().Version())
	a.InternalAttrs = app.InternalAttributes{
		Info:            appInfo,
		Linker:          link.NewRedirectedInternalLinker(appInfo, a.Flag.HubEndpoint),
		BackendIdentity: identity,
		RPCTransport:    identity.HTTPTransport(daemon.HubIdentity.SPIFFEPath()),
		DisableConsole:  true,
		InprocHostPath:  link.InprocHostPath,
	}
}

func (*LinkApp) InitComponents(addComponent app.TypeAdder) {
	addComponent(app.T[*hubinfo.HubInfo]())
	addComponent(app.T[*hubredis.Client]())
	addComponent(app.T[*linknats.Client]())
}

func (*LinkApp) InitModules(addModule app.TypeAdder) {
	addModule(app.T[*config.Reader]())
	addModule(app.T[*event.Manager]())
	addModule(app.T[*task.Manager]())
	addModule(app.T[*rpcproxy.RpcProxy]())
	addModule(app.T[*webproxy.WebProxy]())
	addModule(app.T[*ingress.Ingress]())
	addModule(app.T[*minder.AppMinder]())
}

func (*LinkApp) ServicerInitHandlers(addHandler app.TypeAdder) {
	addHandler(app.T[*impl.BootServiceServerImpl]())
	addHandler(app.T[*impl.RegistryServiceServerImpl]())
	addHandler(app.T[*impl.ConfigServiceServerImpl]())
	addHandler(app.T[*impl.EventServiceServerImpl]())
	addHandler(app.T[*impl.TaskServiceServerImpl]())
}
