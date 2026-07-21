package app

import (
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"net/http"
)

// Module

type Module = Component

type BaseModule = BaseComponent

// PathPrefixRouteModule

type PathPrefixRouteAdder func(prefix string, httpHandler http.Handler, rpcHandler rpcspec.RpcHandler)

type PathPrefixRouteModule interface {
	InitPathPrefixRoute(add PathPrefixRouteAdder)
}
