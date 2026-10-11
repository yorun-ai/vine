package rpcgw

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/desc"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

func inspectService(name string) *skeldesc.Service {
	return new(skeldesc.Service{
		Name: "UserService", SkelName: name, Api: true,
		Audiences: []*skeldesc.ActorAudience{{SkelName: "demo.UserActor"}},
		Methods: []*skeldesc.Method{{Name: "Get", SkelName: "GetV2", EffectiveAuthMode: skeldesc.AuthModeRequired,
			Arguments:  []*skeldesc.Member{{Name: "id", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt})}},
			ResultType: new(skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.User"}),
		}},
	})
}

func inspectWatch(t *testing.T) *watchtest.Client {
	watch := newTestDescriptorWatch(t)
	watch.SetValue(watched.FormatDescriptorServiceKey("demo.UserService"), vcode.MustMarshalJsonS(inspectService("demo.UserService")))
	watch.SetValue(watched.FormatDescriptorDataKey("demo.User"), vcode.MustMarshalJsonS(skeldesc.Data{
		SkelName: "demo.User", Members: []*skeldesc.Member{{Name: "name", Type: new(skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString})}},
	}))
	return watch
}

func inspectConfig(names ...string) watched.PortalSite {
	config := watched.PortalSite{
		Name: "api", Type: "RPCGW", ActorSkelName: "demo.UserActor",
		Cors:        watched.PortalCors{Mode: watched.PortalCorsModeSameDomain},
		RpcgwConfig: new(watched.PortalRpcgwConfig{InspectEnabled: true}),
	}
	for _, name := range names {
		config.RpcgwConfig.Services = append(config.RpcgwConfig.Services, watched.PortalRpcgwService{SkelName: name})
	}
	return config
}

func newInspectGateway(t *testing.T, watch hubwatch.ClientOps, config watched.PortalSite) *RpcGateway {
	t.Helper()
	descriptors := new(desc.Manager{Context: t.Context(), Watch: watch})
	descriptors.DIInit()
	t.Cleanup(descriptors.AfterAppStop)
	endpoints := new(epmgr.Manager{Context: t.Context(), Watch: watch})
	endpoints.DIInit()
	admission := new(access.Access{Context: t.Context(), Desc: descriptors, Epmgr: endpoints})
	admission.DIInit()
	t.Cleanup(admission.AfterAppStop)
	gateway := New(t.Context(), testServerApp(), admission, descriptors, endpoints, config)
	t.Cleanup(gateway.Stop)
	return gateway
}

func inspectRequest(gateway *RpcGateway, method string, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "http://api.example.com"+path, nil)
	request.Header.Set("Origin", "https://console.example.com")
	gateway.Serve(testContextWithEntryOrigin(recorder, request, "api.example.com"))
	return recorder
}

func inspectResult(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.NotEmpty(t, recorder.Header().Get(spec.HeaderPortalTraceId))
	assertInspectHeaders(t, recorder)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), target))
}

func assertInspectHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.Empty(t, recorder.Header().Get("Vrpc-Status"))
	require.Empty(t, recorder.Header().Get("Vrpc-Server"))
}

func assertInspectError(t *testing.T, recorder *httptest.ResponseRecorder, code ex.Code, message string) {
	t.Helper()
	assertInspectHeaders(t, recorder)
	require.Equal(t, ex.HTTPStatusCode(code), recorder.Code)
	require.JSONEq(t, vcode.MustMarshalJsonS(map[string]string{"code": string(code), "message": message}), recorder.Body.String())
}

func TestInspectListsOnlySiteActorAPIsAndUsesWireMethodNames(t *testing.T) {
	watch := inspectWatch(t)
	service := inspectService("demo.UserService")
	service.Description = "Users"
	service.Deprecated = true
	service.DeprecatedReason = "Use demo.UserServiceV2"
	watch.SetValue(watched.FormatDescriptorServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
	for _, name := range []string{"demo.ZService", "demo.Unconfigured", "demo.OtherActor", "demo.Internal"} {
		service := inspectService(name)
		if name == "demo.OtherActor" {
			service.Audiences[0].SkelName = "demo.OtherActor"
		}
		if name == "demo.Internal" {
			service.Api = false
		}
		watch.SetValue(watched.FormatDescriptorServiceKey(name), vcode.MustMarshalJsonS(service))
	}
	gateway := newInspectGateway(t, watch, inspectConfig("demo.ZService", "demo.UserService", "demo.OtherActor", "demo.Internal"))
	for _, path := range []string{"/inspect", "/inspect/", "/inspect/_list"} {
		var services _InspectServiceList
		inspectResult(t, inspectRequest(gateway, http.MethodGet, path), &services)
		require.Len(t, services.Services, 2)
		require.Equal(t, []string{"demo.UserService", "demo.ZService"}, []string{services.Services[0].SkelName, services.Services[1].SkelName})
	}
	var methods _InspectMethodList
	inspectResult(t, inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/_list"), &methods)
	require.Equal(t, _InspectService{
		Name: "UserService", SkelName: "demo.UserService", Description: "Users",
		Deprecated: true, DeprecatedReason: "Use demo.UserServiceV2",
	}, methods.Service)
	require.Len(t, methods.Methods, 1)
	require.Equal(t, "GetV2", methods.Methods[0].SkelName)
	require.Equal(t, skeldesc.AuthModeRequired, methods.Methods[0].AuthMode)
	var detail _InspectDetail
	recorder := inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/GetV2")
	inspectResult(t, recorder, &detail)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &fields))
	require.NotContains(t, fields, "result")
	require.NotContains(t, fields, "error")
	require.Equal(t, "demo.UserService", detail.Service)
	require.Equal(t, "GetV2", detail.SkelName)
	require.Equal(t, "object", detail.InputSchema["type"])
	require.Equal(t, "#/$defs/demo.User", detail.OutputSchema["$ref"])
	user := detail.OutputSchema["$defs"].(map[string]any)["demo.User"].(map[string]any)
	require.Equal(t, "string", user["properties"].(map[string]any)["name"].(map[string]any)["type"])
	require.NotContains(t, recorder.Body.String(), "revision")
	require.Equal(t, "https://console.example.com", recorder.Header().Get(spec.HeaderAccessControlAllowOrigin))
	for _, path := range []string{"/inspect/demo.Internal/_list", "/inspect/demo.OtherActor/_list", "/inspect/demo.Unconfigured/_list", "/inspect/demo.UserService/Get", "/inspect/demo.UserService", "/inspect/demo.UserService/GetV2/extra"} {
		require.Equal(t, http.StatusNotFound, inspectRequest(gateway, http.MethodGet, path).Code, path)
	}
}

func TestInspectSwitchHotReloadAndCORS(t *testing.T) {
	config := inspectConfig("demo.UserService")
	config.RpcgwConfig.InspectEnabled = false
	gateway := newInspectGateway(t, inspectWatch(t), config)
	require.Equal(t, http.StatusNotFound, inspectRequest(gateway, http.MethodGet, "/inspect/_list").Code)
	config.RpcgwConfig.InspectEnabled = true
	require.True(t, gateway.Update(config))
	require.Equal(t, http.StatusOK, inspectRequest(gateway, http.MethodGet, "/inspect/_list").Code)
	preflight := inspectRequest(gateway, http.MethodOptions, "/inspect/_list")
	require.Equal(t, http.StatusNoContent, preflight.Code)
	require.Equal(t, "GET, OPTIONS", preflight.Header().Get(spec.HeaderAccessControlAllowMethods))
	assertInspectError(t, inspectRequest(gateway, http.MethodPost, "/inspect/_list"), ex.InvalidRequest, "inspect requires GET")
	config.ActorSkelName = "demo.OtherActor"
	gateway.Update(config)
	var services _InspectServiceList
	recorder := inspectRequest(gateway, http.MethodGet, "/inspect/_list")
	inspectResult(t, recorder, &services)
	require.JSONEq(t, `{"services":[]}`, recorder.Body.String())
	config.RpcgwConfig.InspectEnabled = false
	gateway.Update(config)
	require.Equal(t, http.StatusNotFound, inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/GetV2").Code)
}

type inspectBlockingWatch struct {
	hubwatch.ClientOps
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *inspectBlockingWatch) LoadKeysAndSubscribe(ctx context.Context, keys []string, handle func(hubwatch.Event)) (map[string]string, hubwatch.Subscription, error) {
	if slices.Contains(keys, watched.FormatDescriptorDataKey("demo.Missing")) {
		w.once.Do(func() { close(w.entered) })
		select {
		case <-w.release:
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
	return w.ClientOps.LoadKeysAndSubscribe(ctx, keys, handle)
}

func TestInspectFailsWhileSyncingAndRecoversMissingTypeFromWatch(t *testing.T) {
	client := inspectWatch(t)
	watch := &inspectBlockingWatch{ClientOps: client, entered: make(chan struct{}), release: make(chan struct{})}
	gateway := newInspectGateway(t, watch, inspectConfig("demo.UserService"))
	service := inspectService("demo.UserService")
	service.Methods[0].ResultType.SkelName = "demo.Missing"
	client.Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorServiceKey(service.SkelName), Value: vcode.MustMarshalJsonS(service)})
	select {
	case <-watch.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("type refresh did not start")
	}
	for _, path := range []string{"/inspect/_list", "/inspect/demo.UserService/_list", "/inspect/demo.UserService/GetV2"} {
		recorder := inspectRequest(gateway, http.MethodGet, path)
		assertInspectError(t, recorder, ex.ServiceUnavailable, "inspect descriptors are syncing")
		require.Equal(t, "1", recorder.Header().Get("Retry-After"))
	}
	close(watch.release)
	require.Eventually(t, func() bool { return !gateway.desc.Snapshot().Syncing }, 3*time.Second, time.Millisecond)
	assertInspectError(t, inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/GetV2"), ex.ServiceUnavailable, "inspect schema is unavailable")
	client.Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorDataKey("demo.Missing"), Value: vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Missing", Description: "Recovered"})})
	require.Eventually(t, func() bool {
		snapshot := gateway.desc.Snapshot()
		return !snapshot.Syncing && snapshot.Data["demo.Missing"] != nil
	}, 3*time.Second, time.Millisecond)
	recorder := inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/GetV2")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Recovered")
	client.Publish(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: watched.FormatDescriptorServiceKey(service.SkelName)})
	require.Eventually(t, func() bool {
		return inspectRequest(gateway, http.MethodGet, "/inspect/demo.UserService/GetV2").Code == http.StatusNotFound
	}, 3*time.Second, time.Millisecond)
}

func TestInspectConcurrentSiteUpdatesKeepSnapshotCoherent(t *testing.T) {
	gateway := newInspectGateway(t, inspectWatch(t), inspectConfig("demo.UserService"))
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			gateway.Update(inspectConfig("demo.UserService"))
			gateway.Update(inspectConfig())
		}
	})
	for range 100 {
		require.Equal(t, http.StatusOK, inspectRequest(gateway, http.MethodGet, "/inspect/_list").Code)
	}
	wg.Wait()
}
