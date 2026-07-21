package rpcproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreapp "go.yorun.ai/vine/internal/core/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

func TestResolveInboundAppStateUsesInstanceID(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	firstApp := mustMetaApp(t, "first.app", "11111111-1111-1111-1111-111111111111")
	secondApp := mustMetaApp(t, "second.app", "22222222-2222-2222-2222-222222222222")

	registerLocalApp(proxy, firstApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.shared"})
	registerLocalApp(proxy, secondApp, "http://127.0.0.1:8081"+testPathRpcInvoke, "http://127.0.0.1:8081", []string{"demo.shared"})

	appState, rpcPath, exErr := proxy.resolveInboundAppState("/" + secondApp.InstanceId() + "/demo.shared/Ping")
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}
	if rpcPath != "/demo.shared/Ping" {
		t.Fatalf("unexpected rpc path: %s", rpcPath)
	}
	if appState.appInfo.InstanceId() != secondApp.InstanceId() {
		t.Fatalf("unexpected app instance: %s", appState.appInfo.InstanceId())
	}
}

func TestResolveInboundAppStateRejectsUnknownService(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	localApp := mustMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")

	registerLocalApp(proxy, localApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.UserService"})

	_, _, exErr := proxy.resolveInboundAppState("/" + localApp.InstanceId() + "/demo.service.OtherService/Ping")
	if exErr == nil {
		t.Fatal("expected resolve error")
	}
	if exErr.Code() != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unexpected error code: %s", exErr.Code())
	}
}

func TestHandleInForwardsHTTPToResolvedInstance(t *testing.T) {
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	localApp := mustMetaApp(t, "local.app", "22222222-2222-2222-2222-222222222222")
	called := false

	targetServer := newH2CTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set(rpchttp.HeaderRpcStatus, string(ex.OK))
		w.Header().Set(rpchttp.HeaderRpcServer, formatAppHeader(localApp))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	proxy := newTestRpcProxy(t, nil)
	registerLocalApp(proxy, localApp, targetServer.URL+testPathRpcInvoke, targetServer.URL, []string{"demo.shared"})

	req := newInboundProxyRequest(t, clientApp, "/"+localApp.InstanceId()+"/demo.shared/Ping")
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	resp := recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if !called {
		t.Fatal("expected target server to be called")
	}
	if resp.Header.Get(rpchttp.HeaderRpcServer) != formatAppHeader(localApp) {
		t.Fatalf("unexpected rpc server header: %s", resp.Header.Get(rpchttp.HeaderRpcServer))
	}
}

func TestHandleInReturnsServiceUnavailableForUnknownInstance(t *testing.T) {
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	proxy := newTestRpcProxy(t, nil)

	req := newInboundProxyRequest(t, clientApp, "/22222222-2222-2222-2222-222222222222/demo.shared/Ping")
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	assertGatewayErrorStatus(t, recorder.Result(), ex.ServiceUnavailable)
}

func TestHandleInReturnsServiceUnavailableForDrainingInstance(t *testing.T) {
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	localApp := mustMetaApp(t, "local.app", "22222222-2222-2222-2222-222222222222")
	proxy := newTestRpcProxy(t, nil)
	registerLocalApp(proxy, localApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.shared"})
	proxy.OnDrain(&minder.AppInstance{AppInfo: localApp})

	req := newInboundProxyRequest(t, clientApp, "/"+localApp.InstanceId()+"/demo.shared/Ping")
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	assertGatewayErrorStatus(t, recorder.Result(), ex.ServiceUnavailable)
}

func TestHandleInReturnsServiceUnavailableForResponseServerMismatch(t *testing.T) {
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	localApp := mustMetaApp(t, "local.app", "22222222-2222-2222-2222-222222222222")
	wrongApp := mustMetaApp(t, "wrong.app", "33333333-3333-3333-3333-333333333333")

	targetServer := newH2CTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(rpchttp.HeaderRpcStatus, string(ex.OK))
		w.Header().Set(rpchttp.HeaderRpcServer, formatAppHeader(wrongApp))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	proxy := newTestRpcProxy(t, nil)
	registerLocalApp(proxy, localApp, targetServer.URL+testPathRpcInvoke, targetServer.URL, []string{"demo.shared"})

	req := newInboundProxyRequest(t, clientApp, "/"+localApp.InstanceId()+"/demo.shared/Ping")
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	assertGatewayErrorStatus(t, recorder.Result(), ex.ServiceUnavailable)
}

func TestHandleInForwardsInprocToResolvedInstance(t *testing.T) {
	methodInfo := ensureTestInboundMethodInfo()
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	localApp := mustMetaApp(t, "local.app", "22222222-2222-2222-2222-222222222222")

	proxy := newTestRpcProxy(t, nil)
	hostPath := coreapp.InprocHostPath(localApp.InstanceId())
	registerLocalApp(proxy, localApp, rpcinproc.Endpoint(hostPath, testPathRpcInvoke), hostPath, []string{methodInfo.Service().SkelName()})

	localState, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state")
	}
	var targetDeadline time.Time
	registerTestInprocHandler(t, localState.serviceEndpoint, spec.RpcHandlerFunc(func(rpcRequest spec.Request) spec.Response {
		targetDeadline, _ = rpcRequest.Context().Deadline()
		if rpcRequest.MethodInfo().Service().SkelName() != methodInfo.Service().SkelName() {
			t.Fatalf("unexpected service: %s", rpcRequest.MethodInfo().Service().SkelName())
		}
		if rpcRequest.MethodInfo().SkelName() != methodInfo.SkelName() {
			t.Fatalf("unexpected method: %s", rpcRequest.MethodInfo().SkelName())
		}
		return &spec.ResponseImpl{
			ServerValue: localApp,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  ex.NewOK(),
		}
	}))

	req := newInboundProxyRequest(t, clientApp, "/"+localApp.InstanceId()+"/"+methodInfo.Service().SkelName()+"/"+methodInfo.SkelName())
	requestDeadline := time.Now().Add(time.Hour)
	ctx, cancel := context.WithDeadline(req.Context(), requestDeadline)
	defer cancel()
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	resp := recorder.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if resp.Header.Get(rpchttp.HeaderRpcStatus) != string(ex.OK) {
		t.Fatalf("unexpected rpc status: %s", resp.Header.Get(rpchttp.HeaderRpcStatus))
	}
	if resp.Header.Get(rpchttp.HeaderRpcServer) != formatAppHeader(localApp) {
		t.Fatalf("unexpected rpc server header: %s", resp.Header.Get(rpchttp.HeaderRpcServer))
	}
	if !targetDeadline.Equal(requestDeadline) {
		t.Fatalf("target deadline = %s, want %s", targetDeadline, requestDeadline)
	}
}

func TestHandleInMapsCancelledInprocInvocationToServiceUnavailable(t *testing.T) {
	methodInfo := ensureTestInboundMethodInfo()
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	localApp := mustMetaApp(t, "local.app", "22222222-2222-2222-2222-222222222222")

	proxy := newTestRpcProxy(t, nil)
	hostPath := coreapp.InprocHostPath(localApp.InstanceId())
	registerLocalApp(proxy, localApp, rpcinproc.Endpoint(hostPath, testPathRpcInvoke), hostPath, []string{methodInfo.Service().SkelName()})

	localState, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state")
	}
	releaseHandler := make(chan struct{})
	registerTestInprocHandler(t, localState.serviceEndpoint, spec.RpcHandlerFunc(func(rpcRequest spec.Request) spec.Response {
		<-releaseHandler
		return &spec.ResponseImpl{
			ServerValue: localApp,
			MethodValue: rpcRequest.MethodInfo(),
			ErrorValue:  ex.NewOK(),
		}
	}))
	defer close(releaseHandler)

	req := newInboundProxyRequest(t, clientApp, "/"+localApp.InstanceId()+"/"+methodInfo.Service().SkelName()+"/"+methodInfo.SkelName())
	reqCtx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(reqCtx)
	recorder := httptest.NewRecorder()

	proxy.handleIn(recorder, req)

	assertGatewayErrorStatus(t, recorder.Result(), ex.ServiceUnavailable)
}
