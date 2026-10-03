// Package linktest supplies a Linker fixture for application tests.
package linktest

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vslice"
)

// Linker records registrations and serves configured values without a Link process.
type Linker struct {
	RpcProxyOutEndpointValue string
	SkipDomainSchemasValue   bool
	LoopbackHostValue        string
	HasLoopbackValue         bool

	RegisterServiceEndpoint string
	RegisterServiceHandlers []linkskeled.ServiceHandlerRegistration
	RegisterWebHandlers     []linkskeled.WebHandlerRegistration
	RegisterEventListeners  []linkskeled.EventListenerRegistration
	RegisterTaskRunners     []linkskeled.TaskRunnerRegistration
	RegisterDomainSchemas   []skel.JSON
	UnregisterCalls         int
	UnregisterError         ex.Error

	EternalConfigByKey map[string]string
	InstantConfigByKey map[string]string
}

func (l *Linker) RpcProxyEndpoint() string {
	return l.RpcProxyOutEndpointValue
}

func (l *Linker) SkipDomainSchemas() bool {
	return l.SkipDomainSchemasValue
}

func (l *Linker) CheckLoopback() (string, bool) {
	return l.LoopbackHostValue, l.HasLoopbackValue
}

func (l *Linker) RegistryClient() linkskeled.RegistryServiceClient {
	return &_TestLinkRegistryClient{linker: l}
}

func (l *Linker) RegistryClientER() linkskeled.RegistryServiceClientER {
	return &_TestLinkRegistryClientER{linker: l}
}

func (l *Linker) ConfigClient() linkskeled.ConfigServiceClient {
	return &_TestLinkConfigClient{linker: l}
}

func (l *Linker) EventClient() linkskeled.EventServiceClient {
	return &_TestLinkEventClient{}
}

func (l *Linker) TaskClient() linkskeled.TaskServiceClient {
	return &_TestLinkTaskClient{}
}

func (l *Linker) LockClient() linkskeled.LockServiceClient {
	return nil
}

type _TestLinkRegistryClient struct {
	linker *Linker
}

func (c *_TestLinkRegistryClient) Register(registration linkskeled.AppRegistration, _ivOpts ...rpcclient.InvokeOption) {
	l := c.linker
	l.RegisterServiceEndpoint = registration.ServiceEndpoint
	l.RegisterServiceHandlers = vslice.Clone(registration.ServiceHandlers)
	l.RegisterWebHandlers = vslice.Clone(registration.WebHandlers)
	l.RegisterEventListeners = vslice.Clone(registration.EventListeners)
	l.RegisterTaskRunners = vslice.Clone(registration.TaskRunners)
	l.RegisterDomainSchemas = append([]skel.JSON(nil), registration.DomainSchemas...)
}

func (c *_TestLinkRegistryClient) Unregister(_ivOpts ...rpcclient.InvokeOption) {
	c.linker.UnregisterCalls++
}

type _TestLinkRegistryClientER struct {
	linker *Linker
}

func (c *_TestLinkRegistryClientER) Register(registration linkskeled.AppRegistration, _ivOpts ...rpcclient.InvokeOption) ex.Error {
	c.linker.RegistryClient().Register(registration, _ivOpts...)
	return nil
}

func (c *_TestLinkRegistryClientER) Unregister(_ivOpts ...rpcclient.InvokeOption) ex.Error {
	c.linker.UnregisterCalls++
	return c.linker.UnregisterError
}

type _TestLinkConfigClient struct {
	linker *Linker
}

func (c *_TestLinkConfigClient) GetEternal(key string, _ivOpts ...rpcclient.InvokeOption) string {
	return c.linker.EternalConfigByKey[key]
}

func (c *_TestLinkConfigClient) GetInstant(key string, _ivOpts ...rpcclient.InvokeOption) string {
	return c.linker.InstantConfigByKey[key]
}

type _TestLinkEventClient struct{}

type _TestLinkTaskClient struct{}

func (*_TestLinkEventClient) EmitEvent(_ linkskeled.EventEmission, _ivOpts ...rpcclient.InvokeOption) {
}

func (*_TestLinkTaskClient) LaunchTask(_ linkskeled.TaskLaunch, _ivOpts ...rpcclient.InvokeOption) {
}

var _ link.Linker = (*Linker)(nil)
