package control

import (
	"cmp"
	"uuid"

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
	s.RegistryCore.Register(toCoreAppRegistration(reg))
	s.refreshDescriptors()
	s.refreshPortalSiteRpcgwServices()
}

func toCoreAppRegistration(reg skeled.AppRegistration) core.AppRegistration {
	domains, hashes := decodeRegisteredDescriptors(reg)
	return core.AppRegistration{
		InstanceId:        reg.InstanceId,
		Name:              reg.Name,
		Version:           reg.Version,
		Endpoint:          reg.Endpoint,
		ServiceHandlers:   toCoreServiceHandlerRegistrations(reg.ServiceHandlers, hashes.services),
		WebHandlers:       toCoreWebHandlerRegistrations(reg.WebHandlers, hashes.webs),
		EventListeners:    toCoreEventListenerRegistrations(reg.EventListeners, hashes.events),
		TaskRunners:       toCoreTaskRunnerRegistrations(reg.TaskRunners, hashes.tasks),
		DomainDescriptors: domains,
	}
}

func (s *RegistryServiceServerImpl) Unregister(name string, instanceId uuid.UUID) {
	s.RegistryCore.Unregister(name, instanceId)
	s.refreshDescriptors()
	s.refreshPortalSiteRpcgwServices()
}

func (s *RegistryServiceServerImpl) Heartbeat(status skeled.AppStatus) bool {
	return s.RegistryCore.Heartbeat(core.AppHeartbeat{
		Name:       status.Name,
		InstanceId: status.InstanceId,
	})
}

func (s *RegistryServiceServerImpl) refreshPortalSiteRpcgwServices() {
	s.Syncer.RefreshPortalSites(s.PortalSiteCore.List)
}

func (s *RegistryServiceServerImpl) refreshDescriptors() {
	s.Syncer.RefreshDescriptors(s.DescriptorRepo.ListDomainDescriptorViews)
}

func toCoreServiceHandlerRegistrations(registrations []skeled.ServiceHandlerRegistration, hashes _LegacyHashes) []core.ServiceHandlerRegistration {
	return vslice.Collect(func(yield func(core.ServiceHandlerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.ServiceHandlerRegistration{
				ServiceSkelName: registration.ServiceSkelName,
				DescriptorHash:  cmp.Or(registration.DescriptorHash, hashes[registration.ServiceSkelName]),
				Endpoint:        registration.Endpoint,
			}) {
				return
			}
		}
	})
}

func toCoreWebHandlerRegistrations(registrations []skeled.WebHandlerRegistration, hashes _LegacyHashes) []core.WebHandlerRegistration {
	return vslice.Collect(func(yield func(core.WebHandlerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.WebHandlerRegistration{
				WebSkelName:    registration.WebSkelName,
				DescriptorHash: cmp.Or(registration.DescriptorHash, hashes[registration.WebSkelName]),
				Endpoint:       registration.Endpoint,
			}) {
				return
			}
		}
	})
}

func toCoreEventListenerRegistrations(registrations []skeled.EventListenerRegistration, hashes _LegacyHashes) []core.EventListenerRegistration {
	return vslice.Collect(func(yield func(core.EventListenerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.EventListenerRegistration{
				EventSkelName:  registration.EventSkelName,
				DescriptorHash: cmp.Or(registration.DescriptorHash, hashes[registration.EventSkelName]),
				TimeoutMs:      registration.TimeoutMs,
				Concurrency:    registration.Concurrency,
				NoRetry:        registration.NoRetry,
			}) {
				return
			}
		}
	})
}

func toCoreTaskRunnerRegistrations(registrations []skeled.TaskRunnerRegistration, hashes _LegacyHashes) []core.TaskRunnerRegistration {
	return vslice.Collect(func(yield func(core.TaskRunnerRegistration) bool) {
		for _, registration := range registrations {
			if !yield(core.TaskRunnerRegistration{
				TaskSkelName:   registration.TaskSkelName,
				DescriptorHash: cmp.Or(registration.DescriptorHash, hashes[registration.TaskSkelName]),
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
