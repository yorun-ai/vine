package web

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

type facadeTestHandler struct {
	defaultFacadeTestWebServer
}

func (h *facadeTestHandler) Routes(r *Router) {
	r.GET("/ping", h.Ping)
}

func (*facadeTestHandler) Ping() {}

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
		ServerType:        reflect.TypeOf((*facadeTestWebServer)(nil)).Elem(),
		DefaultServerType: reflect.TypeFor[*defaultFacadeTestWebServer](),
	})
}

func TestFacadeConstructorsReturnValues(t *testing.T) {
	server := NewServer(Option{
		HandlerTypes: []reflect.Type{reflect.TypeFor[*facadeTestHandler]()},
	})
	if server == nil {
		t.Fatalf("expected web server")
	}

	if NewContainerExecutor(nil, nil) == nil {
		t.Fatalf("expected container executor")
	}
}

func TestNewContextImplementsContext(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/", nil)
	ctx := NewContext(ginCtx, nil, nil, nil, nil)

	if _, ok := any(ctx).(context.Context); !ok {
		t.Fatalf("expected web context to implement context.Context")
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
