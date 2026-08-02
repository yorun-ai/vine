package app

import (
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
)

func init() {
	rpclog.MuteSuccessLog(skeled.RegistryServiceServer.Register)
	rpclog.MuteSuccessLog(skeled.RegistryServiceServer.Heartbeat)
}
