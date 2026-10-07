package control

import (
	skeltype "go.yorun.ai/skel/types"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/util/vslice"
)

type RegistryServiceServerImpl struct {
	skeled.DefaultRegistryServiceServer

	RegistryCore   *core.RegistryCore   `inject:""`
	PortalSiteCore *core.PortalSiteCore `inject:""`
	DescriptorRepo core.DescriptorRepo  `inject:""`
	Syncer         *syncer.Syncer       `inject:""`
}

func (s *RegistryServiceServerImpl) Register(reg skeled.AppRegistration) {
	s.RegistryCore.Register(core.AppRegistration{
		InstanceId:        reg.InstanceId.String(),
		Name:              reg.Name,
		Version:           reg.Version,
		Endpoint:          reg.Endpoint,
		ServiceHandlers:   toCoreServiceHandlerRegistrations(reg.ServiceHandlers),
		WebHandlers:       toCoreWebHandlerRegistrations(reg.WebHandlers),
		EventListeners:    toCoreEventListenerRegistrations(reg.EventListeners),
		TaskRunners:       toCoreTaskRunnerRegistrations(reg.TaskRunners),
		DomainDescriptors: decodeRegisteredDescriptors(reg),
	})
	s.refreshDescriptors()
	s.refreshPortalSiteRpcgwServices()
}

func (s *RegistryServiceServerImpl) Unregister(name string, instanceId skeltype.UUID) {
	s.RegistryCore.Unregister(name, instanceId.String())
	s.refreshDescriptors()
	s.refreshPortalSiteRpcgwServices()
}

func (s *RegistryServiceServerImpl) Heartbeat(status skeled.AppStatus) bool {
	return s.RegistryCore.Heartbeat(core.AppHeartbeat{
		Name:       status.Name,
		InstanceId: status.InstanceId.String(),
	})
}

func (s *RegistryServiceServerImpl) refreshPortalSiteRpcgwServices() {
	for _, site := range s.PortalSiteCore.List() {
		s.Syncer.SyncPortalSite(site)
	}
}

func (s *RegistryServiceServerImpl) refreshDescriptors() {
	s.Syncer.SyncDescriptors(s.DescriptorRepo.ListDomainDescriptorViews())
}

func toCoreServiceHandlerRegistrations(registrations []skeled.ServiceHandlerRegistration) []core.ServiceHandlerRegistration {
	return vslice.Collect(func(yield func(core.ServiceHandlerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.ServiceHandlerRegistration{
				ServiceSkelName: registration.ServiceSkelName,
				DescriptorHash:  registration.DescriptorHash,
				Endpoint:        registration.Endpoint,
			}) {
				return
			}
		}
	})
}

func toCoreWebHandlerRegistrations(registrations []skeled.WebHandlerRegistration) []core.WebHandlerRegistration {
	return vslice.Collect(func(yield func(core.WebHandlerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.WebHandlerRegistration{
				WebSkelName:    registration.WebSkelName,
				DescriptorHash: registration.DescriptorHash,
				Endpoint:       registration.Endpoint,
			}) {
				return
			}
		}
	})
}

func toCoreEventListenerRegistrations(registrations []skeled.EventListenerRegistration) []core.EventListenerRegistration {
	return vslice.Collect(func(yield func(core.EventListenerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.EventListenerRegistration{
				EventSkelName:  registration.EventSkelName,
				DescriptorHash: registration.DescriptorHash,
				TimeoutMs:      registration.TimeoutMs,
				Concurrency:    registration.Concurrency,
				NoRetry:        registration.NoRetry,
			}) {
				return
			}
		}
	})
}

func toCoreTaskRunnerRegistrations(registrations []skeled.TaskRunnerRegistration) []core.TaskRunnerRegistration {
	return vslice.Collect(func(yield func(core.TaskRunnerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.TaskRunnerRegistration{
				TaskSkelName:   registration.TaskSkelName,
				DescriptorHash: registration.DescriptorHash,
				TimeoutMs:      registration.TimeoutMs,
				Concurrency:    registration.Concurrency,
				NoRetry:        registration.NoRetry,
				CronSchedulers: toCoreTaskRunnerCronSchedulers(registration.CronSchedulers),
			}) {
				return
			}
		}
	})
}

func toCoreTaskRunnerCronSchedulers(schedulers []skeled.TaskRunnerCronScheduler) []core.TaskRunnerCronScheduler {
	return vslice.Collect(func(yield func(core.TaskRunnerCronScheduler) bool) {
		for _, scheduler := range schedulers {
			if !yield(core.TaskRunnerCronScheduler{
				TriggerSkelName: scheduler.TriggerSkelName,
				CronExpr:        scheduler.CronExpr,
			}) {
				return
			}
		}
	})
}
