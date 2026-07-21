package rpcproxy

import "testing"

func TestOnSetupTracksAppState(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	localApp := mustMetaApp(t, "demo.app", "11111111-1111-1111-1111-111111111111")

	registerLocalApp(proxy, localApp, "http://127.0.0.1:8080"+testPathRpcInvoke, "http://127.0.0.1:8080", []string{"demo.service.UserService"})

	state, ok := proxy.getAppStateByInstanceID(localApp.InstanceId())
	if !ok {
		t.Fatal("expected local app state to be tracked")
	}
	if state.serviceEndpoint != "http://127.0.0.1:8080"+testPathRpcInvoke {
		t.Fatalf("unexpected service endpoint: %s", state.serviceEndpoint)
	}
	if !state.hasService("demo.service.UserService") {
		t.Fatal("expected service to be indexed on app state")
	}
}
