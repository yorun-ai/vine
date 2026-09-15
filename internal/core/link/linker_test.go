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
			Logger:         logger.New("vine:test"),
			ServerEndpoint: linker.linkBaseEndpoint + coreapp.PathRpcInvoke,
		})
		linker.bootClient = linkskeled.NewBootServiceClient(linkskeled.NewBootServiceClientER(rpcClient))
		linker.registryClientER = linkskeled.NewRegistryServiceClientER(rpcClient)
		linker.registryClient = linkskeled.NewRegistryServiceClient(linker.registryClientER)
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
	assert.NotNil(t, linker.registryClientER)
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

func TestLinkContextBuildsRPCContext(t *testing.T) {
	ctx := newLinkMetaContext(t.Context())

	assert.NotNil(t, ctx.Trace())
	assert.Nil(t, ctx.Initiator())
	assert.Equal(t, meta.ActorTypeAbsent, ctx.Actor().Type())
}
