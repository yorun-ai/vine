package link

import (
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

type TestLinker struct {
	RpcProxyOutEndpointValue string
	SkipDomainSchemasValue   bool
	LoopbackHostValue        string
	HasLoopbackValue         bool

	RegisterConsoleEndpoint   string
	RegisterServiceEndpoint   string
	RegisterWebEndpointPrefix string
	RegisterEventEndpoint     string
	RegisterTaskEndpoint      string
	RegisterServiceHandlers   []linkskeled.ServiceHandlerRegistration
	RegisterWebHandlers       []linkskeled.WebHandlerRegistration
	RegisterEventListeners    []linkskeled.EventListenerRegistration
	RegisterTaskRunners       []linkskeled.TaskRunnerRegistration
	RegisterDomainSchemas     []skel.JSON
	UnregisterCalls           int

	EternalConfigByKey map[string]string
	InstantConfigByKey map[string]string

	EventEmissions []linkskeled.EventEmission
	TaskLaunches   []linkskeled.TaskLaunch
}

func SetNewLinkerForTest(factory func(app meta.App, endpoint string) Linker) func() {
	prev := newLinker
	newLinker = factory

	return func() {
		newLinker = prev
	}
}

func (l *TestLinker) RpcProxyEndpoint() string {
	return l.RpcProxyOutEndpointValue
}

func (l *TestLinker) SkipDomainSchemas() bool {
	return l.SkipDomainSchemasValue
}

func (l *TestLinker) CheckLoopback() (string, bool) {
	return l.LoopbackHostValue, l.HasLoopbackValue
}

func (l *TestLinker) RegistryClient() linkskeled.RegistryServiceClient {
	return &_TestLinkRegistryClient{linker: l}
}

func (l *TestLinker) ConfigClient() linkskeled.ConfigServiceClient {
	return &_TestLinkConfigClient{linker: l}
}

func (l *TestLinker) EventClient() linkskeled.EventServiceClient {
	return &_TestLinkEventClient{linker: l}
}

func (l *TestLinker) TaskClient() linkskeled.TaskServiceClient {
	return &_TestLinkTaskClient{linker: l}
}

type _TestLinkRegistryClient struct {
	linker *TestLinker
}

func (c *_TestLinkRegistryClient) Register(registration linkskeled.AppRegistration, _ivOpts ...rpcclient.InvokeOption) {
	l := c.linker
	l.RegisterConsoleEndpoint = registration.ConsoleEndpoint
	l.RegisterServiceEndpoint = registration.ServiceEndpoint
	l.RegisterWebEndpointPrefix = registration.WebEndpointPrefix
	l.RegisterEventEndpoint = registration.EventEndpoint
	l.RegisterTaskEndpoint = registration.TaskEndpoint
	l.RegisterServiceHandlers = vslice.Clone(registration.ServiceHandlers)
	l.RegisterWebHandlers = vslice.Clone(registration.WebHandlers)
	l.RegisterEventListeners = vslice.Clone(registration.EventListeners)
	l.RegisterTaskRunners = vslice.Clone(registration.TaskRunners)
	l.RegisterDomainSchemas = append([]skel.JSON(nil), registration.DomainSchemas...)
}

func (c *_TestLinkRegistryClient) Unregister(_ivOpts ...rpcclient.InvokeOption) {
	c.linker.UnregisterCalls++
}

type _TestLinkConfigClient struct {
	linker *TestLinker
}

func (c *_TestLinkConfigClient) GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) string {
	return c.linker.EternalConfigByKey[key]
}

func (c *_TestLinkConfigClient) GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) string {
	return c.linker.InstantConfigByKey[key]
}

type _TestLinkEventClient struct {
	linker *TestLinker
}

type _TestLinkTaskClient struct {
	linker *TestLinker
}

func (c *_TestLinkEventClient) EmitEvent(emit linkskeled.EventEmission, _ivOpts ...rpcclient.InvokeOption) {
	c.linker.EventEmissions = append(c.linker.EventEmissions, emit)
}

func (c *_TestLinkTaskClient) LaunchTask(launch linkskeled.TaskLaunch, _ivOpts ...rpcclient.InvokeOption) {
	c.linker.TaskLaunches = append(c.linker.TaskLaunches, launch)
}
