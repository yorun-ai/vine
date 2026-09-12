package web

import (
	"context"
	"reflect"
	"testing"
)

type facadeTestWebServer interface {
	Handler

	mustBeFacadeTestWebServer()
}

type defaultFacadeTestWebServer struct {
}

func (*defaultFacadeTestWebServer) Routes(*Router) {
	panic("method routes is not implemented")
}

func (*defaultFacadeTestWebServer) mustBeFacadeTestWebServer() {}

func init() {
	Register(&WebSpec{
		Name:              "FacadeTestWeb",
		SkelName:          "demo.user.FacadeTestWeb",
		ServerType:        reflect.TypeFor[facadeTestWebServer](),
		DefaultServerType: reflect.TypeFor[*defaultFacadeTestWebServer](),
	})
}

var _ context.Context = Context(nil)

func TestRegisteredWebInfosContainsFacadeRegisteredWeb(t *testing.T) {
	infos := RegisteredWebInfos()
	for _, info := range infos {
		if info.SkelName() == "demo.user.FacadeTestWeb" {
			return
		}
	}
	t.Fatalf("expected facade registered web to appear in RegisteredWebInfos")
}
