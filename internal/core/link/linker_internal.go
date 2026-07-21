package link

import (
	coreapp "go.yorun.ai/vine/internal/core/app"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vpre"
)

type _InternalLinker struct {
	hubEndpoint string
}

func NewInternalLinker(app meta.App) Linker {
	return &_InternalLinker{}
}

func NewRedirectedInternalLinker(app meta.App, redirectEndpoint string) Linker {
	return &_InternalLinker{
		hubEndpoint: redirectEndpoint,
	}
}

func (l *_InternalLinker) RpcProxyEndpoint() string {
	if l.hubEndpoint == "" {
		return ""
	}
	return l.hubEndpoint + coreapp.PathRpcInvoke
}

func (*_InternalLinker) SkipDomainSchemas() bool {
	return false
}

// Internal applications never register themselves to link, so this hook is not used here.
func (*_InternalLinker) CheckLoopback() (string, bool) {
	return "", false
}

func (*_InternalLinker) RegistryClient() linkskeled.RegistryServiceClient {
	vpre.MustNotReach()
	return nil
}

func (*_InternalLinker) ConfigClient() linkskeled.ConfigServiceClient {
	vpre.MustNotReach()
	return nil
}

func (*_InternalLinker) EventClient() linkskeled.EventServiceClient {
	vpre.MustNotReach()
	return nil
}

func (*_InternalLinker) TaskClient() linkskeled.TaskServiceClient {
	vpre.MustNotReach()
	return nil
}
