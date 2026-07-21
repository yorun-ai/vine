package app

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
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
	return "vine.link"
}

func (a *LinkApp) DIInit() {
	a.Flag.Normalize(a.InprocFlag.Enabled)
	a.AppFlag.ListenAddr = a.Flag.APIListen

	appName := a.Name()
	if a.InprocFlag.Enabled {
		appName += "@" + runtime.Application().Name()
	}
	appInfo := meta.MustNewAppWithRandomId(appName, runtime.Application().Version())
	a.InternalAttrs = app.InternalAttributes{
		Info:           appInfo,
		Linker:         link.NewRedirectedInternalLinker(appInfo, a.Flag.HubEndpoint),
		DisableConsole: true,
		InprocHostPath: link.InprocHostPath,
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
