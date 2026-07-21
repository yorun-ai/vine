package app

import (
	"net/http"
	"strings"

	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

type _ServerRoute struct {
	Prefix      string
	HttpHandler http.Handler
	RpcHandler  rpcspec.RpcHandler
}

func (r _ServerRoute) match(path string) bool {
	return path == r.Prefix || strings.HasPrefix(path, r.Prefix+"/")
}

func (r _ServerRoute) serveHTTP(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, r.Prefix)
	if path == "" {
		path = "/"
	}
	next := req.Clone(req.Context())
	next.URL.Path = path
	next.RequestURI = path
	r.HttpHandler.ServeHTTP(w, next)
}

func (a *_AppImpl) appendRoute(prefix string, httpHandler http.Handler, rpcHandler rpcspec.RpcHandler) {
	a.routes = append(a.routes, _ServerRoute{
		Prefix:      prefix,
		HttpHandler: httpHandler,
		RpcHandler:  rpcHandler,
	})
}
