package app

import (
	"go.yorun.ai/vine/internal/core/app/skeled"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
)

func init() {
	rpclog.MuteSuccessLog(skeled.ConsoleServiceServer.Ping)
}
