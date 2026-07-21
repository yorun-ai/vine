package rpcproxy

import (
	"testing"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

func TestResolveOutboundEndpointRoundRobin(t *testing.T) {
	callerApp := mustMetaApp(t, "caller.app", "11111111-1111-1111-1111-111111111111")
	firstApp := mustMetaApp(t, "first.app", "22222222-2222-2222-2222-222222222222")
	secondApp := mustMetaApp(t, "second.app", "33333333-3333-3333-3333-333333333333")
	firstEndpoint := "http://first.invalid/rpc/proxy/in/" + firstApp.InstanceId()
	secondEndpoint := "http://second.invalid/rpc/proxy/in/" + secondApp.InstanceId()

	proxy := newTestRpcProxy(t, newTestHubRedisClient(map[string][]redised.RpcServiceRegistration{
		"demo.service.UserService": {
			{
				ServiceName:   "demo.service.UserService",
				Endpoint:      firstEndpoint,
				AppName:       firstApp.Name(),
				AppVersion:    firstApp.Version(),
				AppInstanceId: firstApp.InstanceId(),
			},
			{
				ServiceName:   "demo.service.UserService",
				Endpoint:      secondEndpoint,
				AppName:       secondApp.Name(),
				AppVersion:    secondApp.Version(),
				AppInstanceId: secondApp.InstanceId(),
			},
		},
	}))

	registerLocalApp(proxy, callerApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.CallerService"})

	firstResolved, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}

	secondResolved, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}

	if firstResolved == secondResolved {
		t.Fatalf("expected round-robin endpoints, got %s and %s", firstResolved, secondResolved)
	}

	resolvedSet := map[string]struct{}{
		firstResolved:  {},
		secondResolved: {},
	}
	if _, ok := resolvedSet[firstEndpoint]; !ok {
		t.Fatalf("expected first endpoint to participate: %s", firstEndpoint)
	}
	if _, ok := resolvedSet[secondEndpoint]; !ok {
		t.Fatalf("expected second endpoint to participate: %s", secondEndpoint)
	}
}

func TestOnDestroyCleansAppAndServiceState(t *testing.T) {
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

	if _, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp); exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}
	if len(proxy.serviceStatesByName) != 1 {
		t.Fatalf("expected one retained service state, got %d", len(proxy.serviceStatesByName))
	}

	proxy.OnDestroy(&minder.AppInstance{AppInfo: callerApp})

	if _, ok := proxy.getAppStateByInstanceID(callerApp.InstanceId()); ok {
		t.Fatal("expected local app state to be removed")
	}
	if len(proxy.serviceStatesByName) != 0 {
		t.Fatalf("expected retained service state to be released, got %d", len(proxy.serviceStatesByName))
	}
}

func TestOnDrainKeepsOutboundSourceState(t *testing.T) {
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
	proxy.OnDrain(&minder.AppInstance{AppInfo: callerApp})

	resolved, exErr := proxy.resolveOutboundEndpoint("demo.service.UserService", callerApp)
	if exErr != nil {
		t.Fatalf("unexpected resolve error: %v", exErr)
	}
	if resolved != remoteEndpoint {
		t.Fatalf("unexpected resolved endpoint: %s", resolved)
	}
}
