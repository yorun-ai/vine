package webproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const (
	testPathConsole   = "/console"
	testPathRpcInvoke = "/rpc/invoke"
	testPathWebAccess = "/web/access"
	testPathEvent     = "/event"
	testPathTask      = "/task"
)

type _WebProxyRegistryServiceClient struct{}

func (*_WebProxyRegistryServiceClient) Register(hubskeled.AppRegistration, ...client.InvokeOption) {}
func (*_WebProxyRegistryServiceClient) Unregister(string, skel.UUID, ...client.InvokeOption)       {}
func (*_WebProxyRegistryServiceClient) Heartbeat(hubskeled.AppStatus, ...client.InvokeOption) bool {
	return true
}

type _H2CTestServer struct {
	URL       string
	Transport http.RoundTripper
}

type _HandlerRoundTripper struct {
	handler http.Handler
}

func (t _HandlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func newH2CTestServer(t *testing.T, handler http.Handler) *_H2CTestServer {
	t.Helper()

	return &_H2CTestServer{
		URL:       "http://test.local",
		Transport: _HandlerRoundTripper{handler: handler},
	}
}

func mustWebProxyMetaApp(t *testing.T, name string, instanceID string) meta.App {
	t.Helper()
	appInfo, err := meta.NewApp(name, "1.0.0", instanceID)
	if err != nil {
		t.Fatalf("meta.NewApp() error = %v", err)
	}
	return appInfo
}

func newTestAppMinder() *minder.AppMinder {
	appInfo, err := meta.NewApp("vine.link", "1.0.0", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil {
		panic(err)
	}
	minder := &minder.AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{HubInprocMode: true},
		App:                   appInfo,
		InprocFlag:            &internalapp.InternalInprocFlag{Enabled: true},
		RegistryServiceClient: &_WebProxyRegistryServiceClient{},
	}
	minder.DIInit()
	return minder
}

func registerLocalWebApp(proxy *WebProxy, appInfo meta.App, webEndpointPrefix string, ingressEndpoint string, webNames []string) {
	webHandlers := make([]skeled.WebHandlerRegistration, 0, len(webNames))
	for _, webSkelName := range webNames {
		webHandlers = append(webHandlers, skeled.WebHandlerRegistration{
			WebSkelName: webSkelName,
		})
	}
	proxy.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:           appInfo,
		WebEndpointPrefix: webEndpointPrefix,
		IngressEndpoint:   ingressEndpoint,
		WebHandlers:       webHandlers,
	})
}
