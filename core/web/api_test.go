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
var _ interface{ BasePath() string } = (*Router)(nil)

func TestRouteAssemblyIsNotExposedThroughFacadeTypes(t *testing.T) {
	for _, kind := range []reflect.Type{reflect.TypeFor[*Router](), reflect.TypeFor[Route]()} {
		for _, name := range []string{"WithBasePath", "WithPrefix", "RoutesWithPrefix"} {
			if _, ok := kind.MethodByName(name); ok {
				t.Errorf("%s exposes internal route assembly method %s", kind, name)
			}
		}
	}
}

func TestRegisteredWebInfosContainsFacadeRegisteredWeb(t *testing.T) {
	infos := RegisteredWebInfos()
	for _, info := range infos {
		if info.SkelName() == "demo.user.FacadeTestWeb" {
			return
		}
	}
	t.Fatalf("expected facade registered web to appear in RegisteredWebInfos")
}
