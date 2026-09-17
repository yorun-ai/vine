package admin

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/internal/util/httputil"
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
// TestServerShutdownEndsAStalledDashboardRequest pins what stops the listener
// while the Dashboard development server holds a request open: Hub cancels the
// requests it proxies, because a development server that does not answer must not
// hold Hub open the way a request Hub serves itself cannot.
func TestServerShutdownEndsAStalledDashboardRequest(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	proxied := make(chan struct{}, 1)
	devServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxied <- struct{}{}
		select {
		case <-release:
		case <-request.Context().Done():
		}
	}))
	t.Cleanup(devServer.Close)
	t.Cleanup(func() { dashboardDevServerURL = "http://localhost:7098" })
	dashboardDevServerURL = devServer.URL
	t.Setenv(dashboardDevProxyEnv, "1")

	originalTimeout := shutdownTimeout
	shutdownTimeout = 5 * time.Second
	t.Cleanup(func() { shutdownTimeout = originalTimeout })

	server := &Server{
		Context:           context.Background(),
		rpcHTTPHandler:    http.NotFoundHandler(),
		dashboardHandler:  dashboardHandler(),
		dashboardDevProxy: dashboardDevProxy(),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.httpServer = httputil.NewServer(listener.Addr().String(), server)
	go func() { _ = server.httpServer.Serve(listener) }()

	// The request stays in flight: the development server never answers it.
	stalled := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String() + "/app/config")
		stalled <- err
		if err == nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-proxied:
	case <-time.After(5 * time.Second):
		t.Fatal("the Dashboard request did not reach the development server")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		server.BeforeAppStop()
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for the stalled Dashboard request")
	}
	<-stalled
}

func TestServerServesAdminAPIAndDashboardBuild(t *testing.T) {
	apiPath := ""
	server := &Server{
		rpcHTTPHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			apiPath = request.URL.Path
			writer.WriteHeader(http.StatusOK)
		}),
		dashboardHandler: dashboardHandler(),
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://hub.local/api/invoke/InfoService/GetInfo", nil))
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
