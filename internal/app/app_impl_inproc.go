package app

import (
	"net/http"
	"strings"

	coreapp "go.yorun.ai/vine/internal/core/app"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	"go.yorun.ai/vine/util/vpre"
)

func (a *_AppImpl) startInprocServer() {
	if !a.hasInprocRoutes() {
		return
	}
	vpre.CheckNotEmpty(a.inprocFlag.HostPath, "application inproc host path is empty")
	for _, route := range a.routes {
		if route.RpcHandler == nil {
			if route.HttpHandler != nil && isWebAccessRoute(route.Prefix) {
				webinproc.Register(webinproc.Endpoint(a.inprocFlag.HostPath, route.Prefix), http.HandlerFunc(route.serveHTTP))
			}
			continue
		}
		rpcinproc.Register(rpcinproc.Endpoint(a.inprocFlag.HostPath, route.Prefix), route.RpcHandler)
	}
}

func (a *_AppImpl) stopInprocServer() {
	if !a.hasInprocRoutes() {
		return
	}
	for _, route := range a.routes {
		if route.RpcHandler == nil {
			if route.HttpHandler != nil && isWebAccessRoute(route.Prefix) {
				webinproc.Unregister(webinproc.Endpoint(a.inprocFlag.HostPath, route.Prefix))
			}
			continue
		}
		rpcinproc.Unregister(rpcinproc.Endpoint(a.inprocFlag.HostPath, route.Prefix))
	}
}

func (a *_AppImpl) hasInprocRoutes() bool {
	for _, route := range a.routes {
		if route.RpcHandler != nil {
			return true
		}
		if route.HttpHandler != nil && isWebAccessRoute(route.Prefix) {
			return true
		}
	}
	return false
}

func isWebAccessRoute(prefix string) bool {
	return prefix == coreapp.PathWebAccess || strings.HasPrefix(prefix, coreapp.PathWebAccess+"/")
}
