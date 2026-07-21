package skeled

import (
	"reflect"

	web "go.yorun.ai/vine/internal/core/web/spec"
)

func init() {
	web.Register(_DashboardWebSpec)
}

// DashboardWebServer Hub Dashboard Web

var _DashboardWebSpec = &web.WebSpec{
	Name:              "DashboardWeb",
	SkelName:          "vine.hub.DashboardWeb",
	Hash:              "0a818f1f",
	ServerType:        reflect.TypeFor[DashboardWebServer](),
	DefaultServerType: reflect.TypeFor[*DefaultDashboardWebServer](),
}

type DashboardWebServer interface {
	web.Handler

	mustBeDashboardWebServer()
}

type DefaultDashboardWebServer struct {
}

func (*DefaultDashboardWebServer) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*DefaultDashboardWebServer) mustBeDashboardWebServer() {}
