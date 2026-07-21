package rpcproxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
