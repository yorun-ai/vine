package admin

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/meta"
	rpcserver "go.yorun.ai/vine/internal/core/rpc/server"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/configaccess"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	impl "go.yorun.ai/vine/internal/daemon/hub/src/server/impl/admin"
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
	request := httptest.NewRequest(http.MethodGet, "http://hub.local/portal/site", nil)
	request.Header.Set("Accept", "text/html")
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected dashboard status code: %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected the embedded Dashboard, got: %s", response.Body.String())
	}
}

func TestAdminApiAudienceAndAuthenticationContract(t *testing.T) {
	for _, domain := range skel.RegisteredDomainDescriptors() {
		if domain.Name != "vine.hub.admin" {
			continue
		}
		require.Len(t, domain.Actors, 1)
		actor := domain.Actors[0]
		require.Equal(t, "vine.hub.admin.AdminActor", actor.SkelName)
		require.Equal(t, []skeldesc.ActorViaKind{skeldesc.ActorViaClient}, actor.Vias)
		require.Nil(t, actor.Auth, "the admin listener has no actor login service")
		require.Len(t, domain.Services, len(HandlerTypes()))
		for _, service := range domain.Services {
			require.True(t, service.Api, service.SkelName)
			require.Len(t, service.Audiences, 1, service.SkelName)
			require.True(t, service.HasAudience(actor.SkelName, skeldesc.ActorViaClient), service.SkelName)
			require.Equal(t, skeldesc.AuthModeOptional, service.AuthMode, service.SkelName)
		}
		return
	}
	t.Fatal("missing generated Hub admin descriptor")
}

func TestAdminApiRemainsCallableFromDashboardWithoutCredentials(t *testing.T) {
	backend := rpcserver.New(rpcserver.Option{
		App:          meta.MustNewApp("vine.hub", "0.0.0", "123e4567-e89b-12d3-a456-426614174001"),
		HandlerTypes: []reflect.Type{reflect.TypeFor[*impl.AdminApiServiceServerImpl]()},
		Executor:     rpcserver.NewDefaultExecutor(rpcserver.With(new(configaccess.Access))),
	})
	server := new(Server{rpcHTTPHandler: backend.HTTPHandler()})
	request := httptest.NewRequest(http.MethodPost, "http://hub.local/api/invoke/vine.hub.admin.AdminApiService/readOnly", strings.NewReader(`{"arguments":null}`))
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	request.Header.Set(rpchttp.HeaderAccept, rpchttp.ContentTypeJson)
	rpchttp.EncodeTraceToHeader(request.Header, meta.InitialTrace())
	rpchttp.EncodeClientToHeader(request.Header, meta.MustNewApp("vine.hub.dashboard", "0.0.1", "123e4567-e89b-12d3-a456-426614174002"))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	require.Equal(t, "OK", response.Header().Get(rpchttp.HeaderRpcStatus), response.Body.String())
	require.Contains(t, response.Body.String(), `"result":false`)
}
