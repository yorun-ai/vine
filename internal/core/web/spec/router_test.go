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
	router := NewRouter(reflect.TypeFor[*_RouterTestHandler](), "")

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
