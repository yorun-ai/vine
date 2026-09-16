package admin

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/mtls"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func TestServerOpensAdminListenerForInprocHub(t *testing.T) {
	prev := listenTCP
	listenTCP = func(_, _ string) (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }
	t.Cleanup(func() { listenTCP = prev })

	for _, tc := range []struct {
		name        string
		adminListen string
		wantHTTP    bool
	}{
		{name: "address declared", adminListen: "127.0.0.1:7099", wantHTTP: true},
		{name: "address omitted", adminListen: "", wantHTTP: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{
				Context:         context.Background(),
				Flag:            &flag.Flag{AdminListen: tc.adminListen},
				InternalRuntime: _InternalRuntimeStub{},
				Identity:        mtls.DisabledIdentity(),
			}

			if err := server.BeforeAppStart(); err != nil {
				t.Fatalf("start admin server: %v", err)
			}
			defer server.BeforeAppStop()

			if gotHTTP := server.httpServer != nil; gotHTTP != tc.wantHTTP {
				t.Fatalf("http listener started = %v, want %v", gotHTTP, tc.wantHTTP)
			}
			if !tc.wantHTTP {
				return
			}
			response, err := http.Get("http://" + server.httpServer.Addr + "/")
			if err != nil {
				t.Fatalf("reach the admin listener: %v", err)
			}
			defer func() { _ = response.Body.Close() }()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("unexpected dashboard status code: %d", response.StatusCode)
			}
		})
	}
}

type _InternalRuntimeStub struct{}

func (_InternalRuntimeStub) AdditionalServicer(...reflect.Type) (http.Handler, rpcspec.RpcHandler) {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("admin api"))
	}), rpcspec.RpcHandlerFunc(func(rpcspec.Request) rpcspec.Response { return nil })
}

// TestServerServesAdminAPIAndDashboardBuild pins the routing of the admin
// listener: the Admin API answers its own path, and every other path serves the
// embedded Dashboard build.
func TestServerServesAdminAPIAndDashboardBuild(t *testing.T) {
	apiPath := ""
	server := &Server{
		rpcHTTPHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			apiPath = request.URL.Path
			writer.WriteHeader(http.StatusOK)
		}),
		dashboardHandler: DashboardHandler(),
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://hub.local/rpc/invoke/InfoService/GetInfo", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected api status code: %d", response.Code)
	}
	if apiPath != "/InfoService/GetInfo" {
		t.Fatalf("unexpected api path: %q", apiPath)
	}

	response = httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hub.local/portal/site", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected dashboard status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected the Dashboard build, got: %s", response.Body.String())
	}
}
