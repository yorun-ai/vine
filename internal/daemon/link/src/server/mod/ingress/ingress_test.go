package ingress

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	internalapp "go.yorun.ai/vine/internal/app"
	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/ex"
	corelink "go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/core/skel"
	webinproc "go.yorun.ai/vine/internal/core/web/inproc"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/rpcproxy"
)

const (
	testPathConsole   = "/console"
	testPathRpcInvoke = "/rpc/invoke"
	testPathWebAccess = "/web/access"
	testPathEvent     = "/event"
	testPathTask      = "/task"
)

type _IngressRegistryServiceClient struct{}

func (*_IngressRegistryServiceClient) Register(hubskeled.AppRegistration, ...client.InvokeOption) {
}
func (*_IngressRegistryServiceClient) Unregister(string, skel.UUID, ...client.InvokeOption) {}
func (*_IngressRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...client.InvokeOption) bool {
	return true
}

func useTestIngressHost(t *testing.T, host string) {
	t.Helper()

	prev := detectHostIP
	detectHostIP = func() string {
		return host
	}
	t.Cleanup(func() {
		detectHostIP = prev
	})
}

func TestIngressServesRpcProxyIn(t *testing.T) {
	localApp := mustMetaApp(t, "demo.app", "22222222-2222-2222-2222-222222222222")
	targetEndpoint := rpcinproc.Endpoint(coreapp.InprocHostPath(localApp.InstanceId()), testPathRpcInvoke)
	rpcinproc.Register(targetEndpoint, spec.RpcHandlerFunc(func(spec.Request) spec.Response {
		return &spec.ResponseImpl{
			ServerValue: localApp,
			ResultValue: "ok",
			ErrorValue:  ex.NewOK(),
		}
	}))
	t.Cleanup(func() {
		rpcinproc.Unregister(targetEndpoint)
	})

	proxy := &rpcproxy.RpcProxy{
		Context:   context.Background(),
		Logger:    logger.NewLogger(logger.GlobalOption()),
		AppMinder: newTestIngressAppMinder(),
	}
	proxy.DIInit()
	inprocEndpoint := coreapp.InprocHostPath(localApp.InstanceId())
	proxy.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:           localApp,
		ConsoleEndpoint:   rpcinproc.Endpoint(inprocEndpoint, testPathConsole),
		ServiceEndpoint:   rpcinproc.Endpoint(inprocEndpoint, testPathRpcInvoke),
		WebEndpointPrefix: webinproc.Endpoint(inprocEndpoint, testPathWebAccess),
		EventEndpoint:     rpcinproc.Endpoint(inprocEndpoint, testPathEvent),
		TaskEndpoint:      rpcinproc.Endpoint(inprocEndpoint, testPathTask),
		IngressEndpoint:   inprocEndpoint,
		ServiceHandlers:   []skeled.ServiceHandlerRegistration{{ServiceSkelName: "vine.app.ConsoleService"}},
	})

	ing := &Ingress{
		Context:  context.Background(),
		Flag:     &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"},
		RpcProxy: proxy,
	}
	ing.DIInit()

	req := newIngressProxyRequest(t, "http://localhost"+rpcproxy.PathIn+"/"+localApp.InstanceId()+"/vine.app.ConsoleService/ping")
	recorder := httptest.NewRecorder()
	ing.httpHandler().ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.StatusCode, string(body))
	}
	if string(body) != `{"result":"ok","error":null}` {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestIngressRegistersInprocEndpointWhenHubInprocModeEnabled(t *testing.T) {
	proxy := &rpcproxy.RpcProxy{
		Context:   context.Background(),
		Logger:    logger.NewLogger(logger.GlobalOption()),
		AppMinder: newTestIngressAppMinder(),
	}
	proxy.DIInit()

	ing := &Ingress{
		Context: context.Background(),
		Flag: &flag.Flag{
			HubInprocMode: true,
		},
		RpcProxy: proxy,
	}
	ing.DIInit()
	ing.AfterAppStart()

	expectedEndpoint := ingressinproc.Endpoint(corelink.InprocHostPath)
	if ing.Endpoint() != expectedEndpoint {
		t.Fatalf("unexpected ingress endpoint: %s", ing.Endpoint())
	}

	req := httptest.NewRequest(http.MethodGet, "/ignored", nil)
	resp, err := ingressinproc.RoundTrip(expectedEndpoint+"/missing", req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	ing.BeforeAppStop()
	if ing.Endpoint() != "" {
		t.Fatalf("expected empty endpoint after stop, got: %s", ing.Endpoint())
	}

	_, err = ingressinproc.RoundTrip(expectedEndpoint+"/missing", req)
	if err == nil {
		t.Fatal("expected inproc endpoint to be unregistered")
	}
}

func TestEndpointOfHostPort(t *testing.T) {
	if got := endpointOfHostPort("127.0.0.1", 18091); got != "http://127.0.0.1:18091" {
		t.Fatalf("unexpected endpoint: %s", got)
	}
}

func mustMetaApp(t *testing.T, name string, instanceID string) meta.App {
	t.Helper()
	appInfo, err := meta.NewApp(name, "1.0.0", instanceID)
	if err != nil {
		t.Fatalf("meta.NewApp() error = %v", err)
	}
	return appInfo
}

func newIngressProxyRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	trace := meta.InitialTrace()
	clientApp := mustMetaApp(t, "remote.app", "33333333-3333-3333-3333-333333333333")
	req.Header.Set("accept", "application/vrpc+json")
	req.Header.Set("content-type", "application/vrpc+json")
	rpchttp.EncodeTraceToHeader(req.Header, trace)
	req.Header.Set(rpchttp.HeaderRpcClient, formatAppHeader(clientApp))
	return req
}

func writeRPCResponse(w http.ResponseWriter, serverApp meta.App, result string) {
	w.Header().Set(rpchttp.HeaderContentType, "application/vrpc+json")
	w.Header().Set(rpchttp.HeaderRpcStatus, string(ex.OK))
	w.Header().Set(rpchttp.HeaderRpcServer, formatAppHeader(serverApp))
	w.WriteHeader(rpchttp.ResponseStatusCode)
	_, _ = w.Write([]byte(`{"result":` + result + `,"error":null}`))
}

func formatAppHeader(appInfo meta.App) string {
	return meta.EncodeAppToDelimited(appInfo)
}

func newTestIngressAppMinder() *minder.AppMinder {
	appInfo, err := meta.NewApp("vine.link", "1.0.0", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil {
		panic(err)
	}
	minder := &minder.AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		App:                   appInfo,
		InprocFlag:            &internalapp.InternalInprocFlag{Enabled: true},
		RegistryServiceClient: &_IngressRegistryServiceClient{},
	}
	minder.DIInit()
	return minder
}
