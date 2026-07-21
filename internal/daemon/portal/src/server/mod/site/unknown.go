package site

import (
	"net/http"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

type _UnknownSite struct {
	name string
	kind string
}

func newUnknownSite(name string, kind string) spec.Site {
	return &_UnknownSite{name: name, kind: kind}
}

func (s *_UnknownSite) Name() string {
	return s.name
}

func (s *_UnknownSite) Serve(ctx *spec.Context) {
	http.Error(ctx.ResponseWriter, "portal site type is not supported: "+s.kind+" ("+s.name+")", http.StatusNotImplemented)
}

func (s *_UnknownSite) Update(config redised.PortalSite) bool {
	return false
}

func (s *_UnknownSite) Stop() {}
