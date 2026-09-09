package rpcproxy

import (
	"context"
	"fmt"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

type _OutboundRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f _OutboundRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestForwardOutboundRequestInheritsContextDeadline(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	requestDeadline := time.Now().Add(time.Hour)
	var targetDeadline time.Time
	proxy.transport = _OutboundRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		targetDeadline, _ = request.Context().Deadline()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	request, err := http.NewRequest(http.MethodPost, "http://link.local/demo.Service/Invoke", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	ctx, cancel := context.WithDeadline(request.Context(), requestDeadline)
	defer cancel()
	request = request.WithContext(ctx)

	response, body, exErr := proxy.forwardOutboundRequest(request, "http://target.local/demo.Service/Invoke")

	if exErr != nil {
		t.Fatalf("unexpected forward error: %v", exErr)
	}
	defer response.Body.Close()
	if string(body) != "ok" {
		t.Fatalf("unexpected response body: %s", body)
	}
	if !targetDeadline.Equal(requestDeadline) {
		t.Fatalf("target deadline = %s, want %s", targetDeadline, requestDeadline)
	}
}

func TestResolveOutboundEndpointPrefersLocalTarget(t *testing.T) {
	callerApp := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
	targetApp := mustMetaApp(t, "target.app", "22222222-2222-2222-2222-222222222222")
	remoteEndpoint := "http://remote.invalid/rpc/proxy/in/" + targetApp.InstanceId()

	proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
		"demo.service.UserService": {{
			ServiceName:   "demo.service.UserService",
			Endpoint:      remoteEndpoint,
			AppName:       targetApp.Name(),
			AppVersion:    targetApp.Version(),
			AppInstanceId: targetApp.InstanceId(),
		}},
	}))

	registerLocalApp(proxy, callerApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.CallerService"})
	registerLocalApp(proxy, targetApp, "http://127.0.0.1:8081"+testPathRpcInvoke, "http://127.0.0.1:8081", []string{"demo.service.UserService"})

	endpoint, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}
	if endpoint != "http://127.0.0.1:8081"+testPathRpcInvoke {
		t.Fatalf("unexpected target endpoint: %s", endpoint)
	}
}

func TestResolveOutboundEndpointFallsBackToRemoteRegistration(t *testing.T) {
	callerApp := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
	remoteApp := mustMetaApp(t, "remote.app", "22222222-2222-2222-2222-222222222222")
	remoteEndpoint := "http://remote.invalid/rpc/proxy/in/" + remoteApp.InstanceId()

	proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
		"demo.service.UserService": {{
			ServiceName:   "demo.service.UserService",
			Endpoint:      remoteEndpoint,
			AppName:       remoteApp.Name(),
			AppVersion:    remoteApp.Version(),
			AppInstanceId: remoteApp.InstanceId(),
		}},
	}))

	registerLocalApp(proxy, callerApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.CallerService"})

	endpoint, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}
	if endpoint != remoteEndpoint {
		t.Fatalf("unexpected remote endpoint: %s", endpoint)
	}
}

func TestResolveOutboundEndpointRejectsLocalTargetWithoutService(t *testing.T) {
	callerApp := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
	targetApp := mustMetaApp(t, "target.app", "22222222-2222-2222-2222-222222222222")
	remoteEndpoint := "http://remote.invalid/rpc/proxy/in/" + targetApp.InstanceId()

	proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
		"demo.service.UserService": {{
			ServiceName:   "demo.service.UserService",
			Endpoint:      remoteEndpoint,
			AppName:       targetApp.Name(),
			AppVersion:    targetApp.Version(),
			AppInstanceId: targetApp.InstanceId(),
		}},
	}))

	registerLocalApp(proxy, callerApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.CallerService"})
	registerLocalApp(proxy, targetApp, "http://127.0.0.1:8081"+testPathRpcInvoke, "http://127.0.0.1:8081", []string{"demo.service.OtherService"})

	endpoint, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr == nil {
		t.Fatal("expected resolve error")
	}
	if endpoint != "" {
		t.Fatalf("expected empty endpoint, got: %s", endpoint)
	}
	if exErr.Code() != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unexpected error code: %s", exErr.Code())
	}
}

func TestApiServiceRejectsBackendCallsAtLink(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(fmt.Sprint(local), func(t *testing.T) {
			caller := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
			target := mustMetaApp(t, "target.app", "22222222-2222-2222-2222-222222222222")
			proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
				"demo.OrderService": {{ServiceName: "demo.OrderService", Api: true, AppName: target.Name(), AppInstanceId: target.InstanceId(), Endpoint: "http://remote.invalid/rpc/proxy/in/target"}},
			}))
			registerLocalApp(proxy, caller, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", nil)
			if local {
				registerLocalApp(proxy, target, "http://127.0.0.1:8081"+testPathRpcInvoke, "http://127.0.0.1:8081", []string{"demo.OrderService"})
			}
			for _, destination := range []string{"", target.Name()} {
				_, err := proxy.resolveOutboundTarget("demo.OrderService", caller, destination)
				if err == nil || err.Code() != ex.ClientForbidden {
					t.Fatalf("API backend call was not rejected: %v", err)
				}
			}
		})
	}
}

func TestOutboundDestinationTransports(t *testing.T) {
	method := ensureTestInboundMethodInfo()
	service := method.Service().SkelName()
	caller := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
	target := mustMetaApp(t, "target.app", "22222222-2222-2222-2222-222222222222")
	proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
		service: {{ServiceName: service, AppName: target.Name(), AppInstanceId: target.InstanceId(), Endpoint: "http://target.invalid"}},
	}))
	registerLocalApp(proxy, caller, "http://caller.invalid", "http://caller.invalid", nil)
	registerLocalApp(proxy, target, "http://target.invalid", "http://target.invalid", []string{service})
	for _, destination := range []string{target.Name(), "missing.app"} {
		t.Run("http/"+destination, func(t *testing.T) {
			called := false
			proxy.transport = _OutboundRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				called = true
				if got := r.Header.Get(rpchttp.HeaderRpcOptions); got != "timeout=10s,future=value" {
					t.Errorf("forwarded options: %s", got)
				}
				if r.URL.Host != "target.invalid" {
					t.Errorf("unexpected target: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
			})
			req := newInboundProxyRequest(t, caller, method.FullURLPath())
			req.Header.Set(rpchttp.HeaderRpcOptions, "timeout=10s,future=value,destination="+destination)
			recorder := httptest.NewRecorder()
			proxy.handleOut(recorder, req)
			if called != (destination == target.Name()) {
				t.Fatalf("forwarded = %v", called)
			}
			if !called {
				assertGatewayErrorStatus(t, recorder.Result(), ex.ServiceUnavailable)
			}
		})
	}
	proxy.AppMinder.UnregisterInstance(target.InstanceId())
	endpoint := "rpc+inproc://destination-test/rpc"
	registerLocalApp(proxy, target, endpoint, "destination-test", []string{service})
	registerTestInprocHandler(t, endpoint, spec.RpcHandlerFunc(func(req spec.Request) spec.Response {
		if req.Destination() != "" {
			t.Error("destination reached server")
		}
		return &spec.ResponseImpl{ServerValue: target, MethodValue: method, ErrorValue: ex.NewOK()}
	}))
	for _, destination := range []string{target.Name(), "missing.app"} {
		response := proxy.serveRpcOut(&spec.RequestImpl{ContextValue: context.Background(), ClientValue: caller, DestinationValue: destination, MethodInfoValue: method})
		want := ex.OK
		if destination != target.Name() {
			want = ex.ServiceUnavailable
		}
		if response.Error().Code() != want {
			t.Fatalf("inproc destination %s: %v", destination, response.Error())
		}
	}
}
