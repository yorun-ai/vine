package app

import (
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
)

func init() {
	rpclog.MuteSuccessLog(appskeled.ConsoleServiceClient.Ping)
	rpclog.MuteSuccessLog(linkskeled.RegistryServiceServer.Register)
	rpclog.MuteSuccessLog(hubskeled.RegistryServiceClient.Register)
	rpclog.MuteSuccessLog(hubskeled.RegistryServiceClient.Heartbeat)
}
