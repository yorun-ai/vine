package impl

import (
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/ingress"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vslice"
)

type RegistryServiceServerImpl struct {
	skeled.DefaultRegistryServiceServer

	Context   spec.Context      `inject:""`
	AppMinder *minder.AppMinder `inject:""`
	Ingress   *ingress.Ingress  `inject:""`
}

func (s *RegistryServiceServerImpl) Register(registration skeled.AppRegistration) {
	s.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:           s.Context.Client(),
		ConsoleEndpoint:   registration.ConsoleEndpoint,
		ServiceEndpoint:   registration.ServiceEndpoint,
		WebEndpointPrefix: registration.WebEndpointPrefix,
		EventEndpoint:     registration.EventEndpoint,
		TaskEndpoint:      registration.TaskEndpoint,
		IngressEndpoint:   s.Ingress.Endpoint(),
		ServiceHandlers:   vslice.Clone(registration.ServiceHandlers),
		WebHandlers:       vslice.Clone(registration.WebHandlers),
		EventListeners:    vslice.Clone(registration.EventListeners),
		TaskRunners:       vslice.Clone(registration.TaskRunners),
		DomainSchemas:     vslice.Clone(registration.DomainSchemas),
	})
}

func (s *RegistryServiceServerImpl) Unregister() {
	s.AppMinder.UnregisterInstance(s.Context.Client().InstanceId())
}
