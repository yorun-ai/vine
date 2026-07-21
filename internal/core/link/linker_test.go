package link

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	coreapp "go.yorun.ai/vine/internal/core/app"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
)

type testRuntimeApp struct {
	name       string
	version    string
	instanceID string
}

func (a testRuntimeApp) Name() string {
	return a.name
}

func (a testRuntimeApp) Version() string {
	return a.version
}

func (a testRuntimeApp) InstanceId() string {
	return a.instanceID
}

func mustHostname(linkBaseEndpoint string) string {
	endpointURL, err := url.Parse(linkBaseEndpoint)
	if err != nil {
		panic(err)
	}
	return endpointURL.Hostname()
}

func TestLinkerReturnsLinkInfoFromEndpoint(t *testing.T) {
	client := &_Linker{
		app:              testRuntimeApp{name: "test.app", version: "1.2.3", instanceID: "00000000-0000-0000-0000-000000000123"},
		linkBaseEndpoint: DefaultEndpoint,
		bootInfo:         linkskeled.BootInfo{RpcProxyEndpointPath: "/rpc/proxy/out"},
	}

	assert.Equal(t, DefaultEndpoint+"/rpc/proxy/out", client.RpcProxyEndpoint())
	assert.False(t, client.SkipDomainSchemas())
}

func TestLinkerBuildsRpcProxyEndpointFromInprocEndpointAndPath(t *testing.T) {
	client := &_Linker{
		app:              testRuntimeApp{name: "test.app", version: "1.2.3", instanceID: "00000000-0000-0000-0000-000000000123"},
		linkBaseEndpoint: InprocEndpoint,
		bootInfo:         linkskeled.BootInfo{RpcProxyEndpointPath: "/rpc/proxy/out"},
	}

	assert.Equal(t, InprocEndpoint+"/rpc/proxy/out", client.RpcProxyEndpoint())
}

func TestLinkerCheckLoopback(t *testing.T) {
	for _, linkBaseEndpoint := range []string{
		"http://127.0.0.1:7079",
		"http://localhost:7079",
		"http://[::1]:7079",
	} {
		client := &_Linker{
			app:              testRuntimeApp{name: "test.app", version: "1.2.3", instanceID: "00000000-0000-0000-0000-000000000123"},
			linkBaseEndpoint: linkBaseEndpoint,
		}

		host, ok := client.CheckLoopback()
		assert.True(t, ok, linkBaseEndpoint)
		assert.Equal(t, mustHostname(linkBaseEndpoint), host)
	}
}

func TestInternalLinkerReturnsEmptyEndpointsWithoutRedirect(t *testing.T) {
	client := NewInternalLinker(testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	})

	assert.Equal(t, "", client.RpcProxyEndpoint())
	assert.False(t, client.SkipDomainSchemas())
}

func TestInternalLinkerReturnsHubRPCInvokeEndpointWhenRedirectSet(t *testing.T) {
	client := NewRedirectedInternalLinker(testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}, "http://demo.local:7071")

	assert.Equal(t, "http://demo.local:7071/rpc/invoke", client.RpcProxyEndpoint())
	assert.False(t, client.SkipDomainSchemas())
}

func TestLinkerSkipDomainSchemas(t *testing.T) {
	client := &_Linker{
		app:              testRuntimeApp{name: "test.app", version: "1.2.3", instanceID: "00000000-0000-0000-0000-000000000123"},
		linkBaseEndpoint: DefaultEndpoint,
		bootInfo:         linkskeled.BootInfo{SkipDomainSchemas: true},
	}

	assert.True(t, client.SkipDomainSchemas())
}

func TestInternalLinkerCheckLoopbackReturnsFalseForNonLocalEndpoint(t *testing.T) {
	client := NewRedirectedInternalLinker(testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}, "http://demo.local:7071")

	host, ok := client.CheckLoopback()
	assert.False(t, ok)
	assert.Empty(t, host)
}

func TestNewLinkerBuildsRPCClients(t *testing.T) {
	app := testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}

	oldFactory := newLinker
	newLinker = func(app meta.App, linkBaseEndpoint string) Linker {
		linker := &_Linker{
			app:              app,
			linkBaseEndpoint: linkBaseEndpoint,
			bootInfo:         linkskeled.BootInfo{RpcProxyEndpointPath: "/rpc/proxy/out"},
		}
		rpcClient := client.New(client.Option{
			Context:        newLinkMetaContext(context.Background()),
			ClientApp:      linker.app,
			Logger:         logger.NewLogger(logger.GlobalOption()),
			ServerEndpoint: linker.linkBaseEndpoint + coreapp.PathRpcInvoke,
		})
		linker.bootClient = linkskeled.NewBootServiceClient(linkskeled.NewBootServiceClientER(rpcClient))
		linker.registryClient = linkskeled.NewRegistryServiceClient(linkskeled.NewRegistryServiceClientER(rpcClient))
		linker.configClient = linkskeled.NewConfigServiceClient(linkskeled.NewConfigServiceClientER(rpcClient))
		linker.eventClient = linkskeled.NewEventServiceClient(linkskeled.NewEventServiceClientER(rpcClient))
		linker.taskClient = linkskeled.NewTaskServiceClient(linkskeled.NewTaskServiceClientER(rpcClient))
		return linker
	}
	t.Cleanup(func() {
		newLinker = oldFactory
	})

	client := NewLinker(app, false, "")
	linker := client.(*_Linker)

	assert.NotNil(t, linker.bootClient)
	assert.NotNil(t, linker.registryClient)
	assert.NotNil(t, linker.configClient)
	assert.NotNil(t, linker.eventClient)
	assert.NotNil(t, linker.taskClient)
}

func TestNewLinkerUsesExplicitEndpoint(t *testing.T) {
	app := testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}

	oldFactory := newLinker
	var actualLinkBaseEndpoint string
	newLinker = func(app meta.App, linkBaseEndpoint string) Linker {
		actualLinkBaseEndpoint = linkBaseEndpoint
		return &_Linker{app: app, linkBaseEndpoint: linkBaseEndpoint}
	}
	t.Cleanup(func() {
		newLinker = oldFactory
	})

	NewLinker(app, false, "http://10.0.0.8:7079")

	assert.Equal(t, "http://10.0.0.8:7079", actualLinkBaseEndpoint)
}

func TestNewLinkerUsesRpcInprocEndpointWhenInprocEnabled(t *testing.T) {
	app := testRuntimeApp{
		name:       "test.app",
		version:    "1.2.3",
		instanceID: "00000000-0000-0000-0000-000000000123",
	}

	oldFactory := newLinker
	var actualLinkBaseEndpoint string
	newLinker = func(app meta.App, linkBaseEndpoint string) Linker {
		actualLinkBaseEndpoint = linkBaseEndpoint
		return &_Linker{app: app, linkBaseEndpoint: linkBaseEndpoint}
	}
	t.Cleanup(func() {
		newLinker = oldFactory
	})

	NewLinker(app, true, "http://10.0.0.8:7079")

	assert.Equal(t, "rpc+inproc://vine/link", actualLinkBaseEndpoint)
}

func TestLinkerRegistryClientReturnsClient(t *testing.T) {
	client := &TestLinker{}

	client.RegistryClient().Register(linkskeled.AppRegistration{
		ServiceEndpoint: "http://127.0.0.1:12345/rpc/invoke",
		ServiceHandlers: []linkskeled.ServiceHandlerRegistration{{
			ServiceSkelName: "demo",
		}},
		WebHandlers: []linkskeled.WebHandlerRegistration{{
			WebSkelName: "default@test.app",
		}},
	})

	assert.Equal(t, "http://127.0.0.1:12345/rpc/invoke", client.RegisterServiceEndpoint)
	assert.Equal(t, []linkskeled.ServiceHandlerRegistration{{ServiceSkelName: "demo"}}, client.RegisterServiceHandlers)
	assert.Equal(t, []linkskeled.WebHandlerRegistration{{WebSkelName: "default@test.app"}}, client.RegisterWebHandlers)
}

func TestLinkerRegistryClientSupportsUnregister(t *testing.T) {
	client := &TestLinker{}

	client.RegistryClient().Unregister()

	assert.Equal(t, 1, client.UnregisterCalls)
}

func TestLinkerConfigClientReturnsClient(t *testing.T) {
	client := &TestLinker{
		EternalConfigByKey: map[string]string{
			"a": `{"a":1}`,
		},
		InstantConfigByKey: map[string]string{
			"b": `{"b":2}`,
		},
	}

	assert.Equal(t, `{"a":1}`, client.ConfigClient().GetEternal("a"))
	assert.Equal(t, `{"b":2}`, client.ConfigClient().GetInstant("b"))
}

func TestLinkerEventClientReturnsClient(t *testing.T) {
	client := &TestLinker{}

	client.EventClient().EmitEvent(linkskeled.EventEmission{
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":1}`,
	})

	assert.Len(t, client.EventEmissions, 1)
	assert.Equal(t, "demo.user.UserCreatedEvent", client.EventEmissions[0].EventSkelName)
	assert.Equal(t, `{"userId":1}`, client.EventEmissions[0].EventJson)
}

func TestLinkerTaskClientReturnsClient(t *testing.T) {
	client := &TestLinker{}

	client.TaskClient().LaunchTask(linkskeled.TaskLaunch{
		TaskSkelName:    "demo.user.RebuildUserIndexTask",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{"groupId":1}`,
	})

	assert.Len(t, client.TaskLaunches, 1)
	assert.Equal(t, "demo.user.RebuildUserIndexTask", client.TaskLaunches[0].TaskSkelName)
	assert.Equal(t, "forGroup", client.TaskLaunches[0].TriggerSkelName)
	assert.Equal(t, `{"groupId":1}`, client.TaskLaunches[0].ArgumentsJson)
}

func TestLinkContextBuildsRPCContext(t *testing.T) {
	ctx := newLinkMetaContext(t.Context())

	assert.NotNil(t, ctx.Trace())
	assert.Nil(t, ctx.Initiator())
	assert.Equal(t, meta.ActorTypeAbsent, ctx.Actor().Type())
}
