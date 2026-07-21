package spec

import (
	"net/http"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

type Context struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	RemoteAddr     string
	EntryOrigin    EntryOrigin
}

type EntryOrigin struct {
	Scheme Scheme
	Host   string
	Port   int
}

type Scheme string

const (
	SchemeHTTP  Scheme = "http"
	SchemeHTTPS Scheme = "https"
)

type Site interface {
	Name() string
	Serve(ctx *Context)
	Update(config redised.PortalSite) bool
	Stop()
}
