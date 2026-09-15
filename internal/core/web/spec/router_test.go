package spec

import (
	"net/http"
	"reflect"
	"testing"
)

type _RouterTestHandler struct{}

func (*_RouterTestHandler) Routes(*Router) {}

func (*_RouterTestHandler) Proxy() {}

func TestRouterANYRegistersCommonMethods(t *testing.T) {
	router := NewRouter(reflect.TypeFor[*_RouterTestHandler](), "/")

	router.ANY("/*path", (&_RouterTestHandler{}).Proxy)

	wantMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
	}
	if len(router.routes) != len(wantMethods) {
		t.Fatalf("expected %d routes, got %d", len(wantMethods), len(router.routes))
	}
	for idx, wantMethod := range wantMethods {
		route := router.routes[idx]
		if route.Method() != wantMethod {
			t.Fatalf("route[%d] method = %s, want %s", idx, route.Method(), wantMethod)
		}
		if route.Path() != "/*path" {
			t.Fatalf("route[%d] path = %s", idx, route.Path())
		}
	}
}

func TestCollectRoutesPreservesNestedPathsAndRegistrations(t *testing.T) {
	router := NewRouter(reflect.TypeFor[*_RouterTestHandler](), "/")
	handler := new(_RouterTestHandler)
	router.GET("/", handler.Proxy)
	child := router.SubRouter("/orders")
	child.GET("/:id", handler.Proxy)
	grandchild := child.SubRouter("/:id")
	grandchild.GET("/items/*path", handler.Proxy)
	if router.BasePath() != "/" || child.BasePath() != "/orders" || grandchild.BasePath() != "/orders/:id" {
		t.Fatalf("unexpected base paths: %q, %q, %q", router.BasePath(), child.BasePath(), grandchild.BasePath())
	}
	if child != router.SubRouter("/orders") || grandchild != child.SubRouter("/:id") {
		t.Fatal("repeated SubRouter calls must reuse the router")
	}

	for _, prefix := range []string{"/demo.OrderWeb", "/demo.OtherWeb"} {
		routes := CollectRoutes(router, prefix)
		wantPaths := []string{prefix + "/", prefix + "/orders/:id", prefix + "/orders/:id/items/*path"}
		if len(routes) != len(wantPaths) {
			t.Fatalf("got %d routes, want %d", len(routes), len(wantPaths))
		}
		for i, want := range wantPaths {
			if routes[i].Path() != want {
				t.Fatalf("route[%d] path = %q, want %q", i, routes[i].Path(), want)
			}
			if routes[i].Method() != http.MethodGet || routes[i].HandlerType() != reflect.TypeFor[*_RouterTestHandler]() || routes[i].HandlerMethod().Name != "Proxy" {
				t.Fatalf("route[%d] lost handler metadata", i)
			}
		}
	}
	if router.Routes()[0].Path() != "/" || child.Routes()[0].Path() != "/:id" {
		t.Fatal("collecting routes changed registered paths")
	}
	if grandchild.BasePath() != "/orders/:id" {
		t.Fatal("collecting routes changed the base path")
	}
}

func TestRouterRejectsDotSegments(t *testing.T) {
	router := NewRouter(reflect.TypeFor[*_RouterTestHandler](), "/")
	for _, register := range []func(){
		func() { router.SubRouter("/../admin") },
		func() { router.GET("/./health", (&_RouterTestHandler{}).Proxy) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("expected dot segment registration to panic")
				}
			}()
			register()
		}()
	}
}
