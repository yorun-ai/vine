package impl

import (
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
)

type BootServiceServerImpl struct {
	skeled.DefaultBootServiceServer

	Flag       *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`
}

func (s *BootServiceServerImpl) GetInfo() skeled.BootInfo {
	return skeled.BootInfo{
		RpcProxyEndpointPath: rpcproxy.PathOut,
		SkipDomainSchemas:    s.callerSharesDomainSchemaRegistryWithHub(),
	}
}

// callerSharesDomainSchemaRegistryWithHub reports whether the application
// calling this inproc Link and the inproc Hub run in the same process. An
// inproc Link can only be reached by applications in its process, while
// HubInprocMode means that Link reaches Hub in that process as well. Only when
// both conditions hold can Hub read the application's schemas from the shared
// process-wide registry instead of receiving them during registration.
func (s *BootServiceServerImpl) callerSharesDomainSchemaRegistryWithHub() bool {
	return s.InprocFlag.Enabled && s.Flag.HubInprocMode
}
