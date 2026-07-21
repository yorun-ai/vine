package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRouteServeHTTPTrimsPrefixToRoot(t *testing.T) {
	var gotPath string
	route := _ServerRoute{
		Prefix: "/web/access/admin@demo.app",
		HttpHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			gotPath = req.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/web/access/admin@demo.app", nil)
	resp := httptest.NewRecorder()

	route.serveHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if gotPath != "/" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}

func TestServerRouteServeHTTPTrimsPrefixToSubPath(t *testing.T) {
	var gotPath string
	route := _ServerRoute{
		Prefix: "/web/access/admin@demo.app",
		HttpHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			gotPath = req.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/web/access/admin@demo.app/ping", nil)
	resp := httptest.NewRecorder()

	route.serveHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if gotPath != "/ping" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}
