package rpcproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
)

const (
	testPathConsole   = "/console"
	testPathRpcInvoke = "/rpc/invoke"
	testPathWebAccess = "/web/access"
	testPathEvent     = "/event"
	testPathTask      = "/task"
)

type _TestRpcProxyInboundServer struct{}
type _TestRpcProxyInboundERServer struct{}

var testRpcProxyInboundServiceOnce sync.Once
var testRpcProxyInboundServiceSpec = &spec.ServiceSpec{
	Type:                spec.ServiceSpecTypeServer,
	Name:                "TestRpcProxyInboundService",
	SkelName:            "rpcproxy.test",
	ServerType:          reflect.TypeFor[*_TestRpcProxyInboundServer](),
	DefaultServerType:   reflect.TypeFor[*_TestRpcProxyInboundServer](),
	ERServerType:        reflect.TypeFor[*_TestRpcProxyInboundERServer](),
	DefaultERServerType: reflect.TypeFor[*_TestRpcProxyInboundERServer](),
	Methods: []*spec.MethodSpec{{
		Name:     "Ping",
		SkelName: "Ping",
	}},
}

type _H2CTestServer struct {
	URL string
}

type _TestRegistryServiceClient struct{}

func (*_TestRegistryServiceClient) Register(hubskeled.AppRegistration, ...client.InvokeOption) {}
func (*_TestRegistryServiceClient) Unregister(string, skel.UUID, ...client.InvokeOption)       {}
func (*_TestRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...client.InvokeOption) bool {
	return true
}

func newTestRpcProxy(t *testing.T, redisClient *hubredis.Client) *RpcProxy {
	t.Helper()
	if redisClient == nil {
		redisClient = hubredis.NewClientForTest(nil)
	}
	minder := &minder.AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		App:                   mustMetaApp(t, "proxy.app", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		InprocFlag:            &internalapp.InternalInprocFlag{Enabled: true},
		RegistryServiceClient: &_TestRegistryServiceClient{},
	}
	minder.DIInit()

	proxy := &RpcProxy{
		Context:     context.Background(),
		RedisClient: redisClient,
		App:         mustMetaApp(t, "proxy.app", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Logger:      logger.NewLogger(logger.GlobalOption()),
		AppMinder:   minder,
	}
	proxy.DIInit()
	return proxy
}

func registerLocalApp(proxy *RpcProxy, appInfo meta.App, serviceEndpoint string, ingressEndpoint string, serviceSkelNames []string) {
	serviceHandlers := make([]skeled.ServiceHandlerRegistration, 0, len(serviceSkelNames))
	for _, serviceSkelName := range serviceSkelNames {
		serviceHandlers = append(serviceHandlers, skeled.ServiceHandlerRegistration{
			ServiceSkelName: serviceSkelName,
		})
	}
	proxy.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:         appInfo,
		ServiceEndpoint: serviceEndpoint,
		IngressEndpoint: ingressEndpoint,
		ServiceHandlers: serviceHandlers,
	})
}

func newTestHubRedisClient(serviceEndpointsByName map[string][]redised.RpcServiceRegistration) *hubredis.Client {
	valuesByKey := map[string]string{}
	for _, registrations := range serviceEndpointsByName {
		for _, registration := range registrations {
			key := redised.FormatRpcServiceRegistrationKey(registration.ServiceName, registration.AppName, registration.AppInstanceId)
			valuesByKey[key] = vcode.MustMarshalJsonS(registration)
		}
	}
	return hubredis.NewClientForTest(valuesByKey)
}

func mustMetaApp(t *testing.T, name string, instanceID string) meta.App {
	t.Helper()
	appInfo, err := meta.NewApp(name, "1.0.0", instanceID)
	if err != nil {
		t.Fatalf("meta.NewApp() error = %v", err)
	}
	return appInfo
}

func formatAppHeader(appInfo meta.App) string {
	return meta.EncodeAppToDelimited(appInfo)
}

func ensureTestInboundMethodInfo() spec.MethodInfo {
	testRpcProxyInboundServiceOnce.Do(func() {
		spec.Register(testRpcProxyInboundServiceSpec)
	})
	return testRpcProxyInboundServiceSpec.Methods[0].Info()
}

func newInboundProxyRequest(t *testing.T, clientApp meta.App, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://proxy.test"+path, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	trace := meta.InitialTrace()
	req.Header.Set(rpchttp.HeaderAccept, "application/vrpc+json")
	req.Header.Set(rpchttp.HeaderContentType, "application/vrpc+json")
	rpchttp.EncodeTraceToHeader(req.Header, trace)
	req.Header.Set(rpchttp.HeaderRpcClient, formatAppHeader(clientApp))
	return req
}

func registerTestInprocHandler(t *testing.T, endpoint string, handler spec.RpcHandler) {
	t.Helper()
	rpcinproc.Register(endpoint, handler)
	t.Cleanup(func() {
		rpcinproc.Unregister(endpoint)
	})
}

func assertGatewayErrorStatus(t *testing.T, resp *http.Response, want ex.Code) {
	t.Helper()
	got := resp.Header.Get(rpchttp.HeaderRpcStatus)
	if got != string(want) {
		t.Fatalf("unexpected rpc status: got %s want %s", got, want)
	}
}

func newH2CTestServer(t *testing.T, handler http.Handler) *_H2CTestServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	server := &http.Server{
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
	})
	port := listener.Addr().(*net.TCPAddr).Port
	return &_H2CTestServer{
		URL: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
}
