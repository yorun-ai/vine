package link

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	rpclog "go.yorun.ai/vine/internal/core/rpc/log"
)

func init() {
	rpclog.MuteSuccessLog(skeled.RegistryServiceClient.Register)
}
