package rpcgw

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

func (g *RpcGateway) serveInspect(ctx *spec.Context) {
	g.writeError(ctx.ResponseWriter, ctx.Request, ex.InvalidRequest, "rpcgw inspect is not implemented")
}
