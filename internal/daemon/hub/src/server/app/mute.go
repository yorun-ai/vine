package app

import (
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
)

func init() {
	rpclog.MuteSuccessLog(skeled.RegistryServiceServer.Register)
	rpclog.MuteSuccessLog(skeled.RegistryServiceServer.Heartbeat)
}
