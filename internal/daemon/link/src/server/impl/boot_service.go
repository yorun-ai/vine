package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
)

type BootServiceServerImpl struct {
	skeled.DefaultBootServiceServer

	Flag *flag.Flag `inject:""`
}

func (s *BootServiceServerImpl) GetInfo() skeled.BootInfo {
	return skeled.BootInfo{
		RpcProxyEndpointPath: rpcproxy.PathOut,
		SkipDomainSchemas:    s.Flag.HubInprocMode,
	}
}
