package link

import (
	"context"
	"net"
	"net/url"

	coreapp "go.yorun.ai/vine/internal/core/app"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	rpcinproc "go.yorun.ai/vine/internal/core/rpc/transport/inproc"
)

const (
	InprocHostPath  = "vine/link"
	InprocEndpoint  = rpcinproc.EndpointScheme + InprocHostPath
	DefaultEndpoint = "http://127.0.0.1:7079"
)

type Linker interface {
	// CheckLoopback reports which loopback host app registration should reuse,
	// and whether this link linkBaseEndpoint is using a loopback host at all.
	CheckLoopback() (string, bool)
	SkipDomainSchemas() bool
	RpcProxyEndpoint() string
	RegistryClient() linkskeled.RegistryServiceClient
	ConfigClient() linkskeled.ConfigServiceClient
	EventClient() linkskeled.EventServiceClient
	TaskClient() linkskeled.TaskServiceClient
}

// Linker

type _Linker struct {
	app              meta.App
	linkBaseEndpoint string

	bootInfo       linkskeled.BootInfo
	bootClient     linkskeled.BootServiceClient
	registryClient linkskeled.RegistryServiceClient
	configClient   linkskeled.ConfigServiceClient
	eventClient    linkskeled.EventServiceClient
	taskClient     linkskeled.TaskServiceClient
}

// newLinker is replaceable in tests so app startup can observe linker calls
// without connecting to a real link process.
var newLinker = func(app meta.App, linkBaseEndpoint string) Linker {
	linker := &_Linker{
		app:              app,
		linkBaseEndpoint: linkBaseEndpoint,
	}
	linker.init()
	return linker
}

func NewLinker(app meta.App, isInproc bool, linkEndpoint string) Linker {
	if isInproc {
		return newLinker(app, InprocEndpoint)
	}
	if linkEndpoint != "" {
		return newLinker(app, linkEndpoint)
	}
	return newLinker(app, DefaultEndpoint)
}

func (l *_Linker) init() {
	rpcClient := client.New(client.Option{
		Context:        newLinkMetaContext(context.Background()),
		ClientApp:      l.app,
		Logger:         logger.NewGlobalLogger(),
		ServerEndpoint: l.linkBaseEndpoint + coreapp.PathRpcInvoke,
	})
	l.bootClient = linkskeled.NewBootServiceClient(linkskeled.NewBootServiceClientER(rpcClient))
	l.registryClient = linkskeled.NewRegistryServiceClient(linkskeled.NewRegistryServiceClientER(rpcClient))
	l.configClient = linkskeled.NewConfigServiceClient(linkskeled.NewConfigServiceClientER(rpcClient))
	l.eventClient = linkskeled.NewEventServiceClient(linkskeled.NewEventServiceClientER(rpcClient))
	l.taskClient = linkskeled.NewTaskServiceClient(linkskeled.NewTaskServiceClientER(rpcClient))
	l.bootInfo = l.bootClient.GetInfo()
}

func newLinkMetaContext(ctx context.Context) meta.Context {
	trace := meta.InitialTrace()
	initiator := meta.Initiator(nil)
	actor := meta.NewAbsentActor()
	return meta.NewContext(ctx, trace, initiator, actor)
}

func (l *_Linker) RpcProxyEndpoint() string {
	return l.linkBaseEndpoint + l.bootInfo.RpcProxyEndpointPath
}

func (l *_Linker) SkipDomainSchemas() bool {
	return l.bootInfo.SkipDomainSchemas
}

func (l *_Linker) CheckLoopback() (string, bool) {
	endpointURL, err := url.Parse(l.linkBaseEndpoint)
	if err != nil {
		return "", false
	}
	host := endpointURL.Hostname()
	if host == "localhost" {
		return host, true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return host, true
	}
	return "", false
}

func (l *_Linker) RegistryClient() linkskeled.RegistryServiceClient {
	return l.registryClient
}

func (l *_Linker) ConfigClient() linkskeled.ConfigServiceClient {
	return l.configClient
}

func (l *_Linker) EventClient() linkskeled.EventServiceClient {
	return l.eventClient
}

func (l *_Linker) TaskClient() linkskeled.TaskServiceClient {
	return l.taskClient
}
