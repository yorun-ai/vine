package impl

import (
	"cmp"

	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type AppStatusServiceServerImpl struct {
	skeled.DefaultAppStatusServiceServer

	RegistryRepo core.RegistryRepo `inject:""`
}

func (s *AppStatusServiceServerImpl) List() []skeled.AppStatusView {
	statuses := s.RegistryRepo.ListAppStatuses()
	items := make([]skeled.AppStatusView, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, toAppStatusView(status))
	}
	return vslice.SortBy(items, func(a skeled.AppStatusView, b skeled.AppStatusView) bool {
		if a.Name != b.Name {
			return cmp.Compare(a.Name, b.Name) < 0
		}
		return cmp.Compare(a.InstanceId, b.InstanceId) < 0
	})
}

func toAppStatusView(status *core.AppStatus) skeled.AppStatusView {
	return skeled.AppStatusView{
		Name:            status.Name,
		InstanceId:      status.InstanceId,
		Version:         status.Version,
		Endpoint:        status.Endpoint,
		ServiceHandlers: toSkeledServiceHandlerRegistrations(status.ServiceHandlers),
		WebHandlers:     toSkeledWebHandlerRegistrations(status.WebHandlers),
		EventListeners:  toSkeledEventListenerRegistrations(status.EventListeners),
		TaskRunners:     toSkeledTaskRunnerRegistrations(status.TaskRunners),
	}
}

func toSkeledServiceHandlerRegistrations(registrations []core.ServiceHandlerRegistration) []skeled.ServiceHandlerRegistration {
	items := []skeled.ServiceHandlerRegistration{}
	for _, registration := range registrations {
		items = append(items, skeled.ServiceHandlerRegistration{
			ServiceSkelName: registration.ServiceSkelName,
			SchemaHash:      registration.SchemaHash,
			Endpoint:        registration.Endpoint,
		})
	}
	return vslice.SortBy(items, func(a skeled.ServiceHandlerRegistration, b skeled.ServiceHandlerRegistration) bool {
		return cmp.Compare(a.ServiceSkelName, b.ServiceSkelName) < 0
	})
}

func toSkeledWebHandlerRegistrations(registrations []core.WebHandlerRegistration) []skeled.WebHandlerRegistration {
	items := []skeled.WebHandlerRegistration{}
	for _, registration := range registrations {
		items = append(items, skeled.WebHandlerRegistration{
			WebSkelName: registration.WebSkelName,
			SchemaHash:  registration.SchemaHash,
			Endpoint:    registration.Endpoint,
		})
	}
	return vslice.SortBy(items, func(a skeled.WebHandlerRegistration, b skeled.WebHandlerRegistration) bool {
		return cmp.Compare(a.WebSkelName, b.WebSkelName) < 0
	})
}

func toSkeledEventListenerRegistrations(registrations []core.EventListenerRegistration) []skeled.EventListenerRegistration {
	items := []skeled.EventListenerRegistration{}
	for _, registration := range registrations {
		items = append(items, skeled.EventListenerRegistration{
			EventSkelName: registration.EventSkelName,
			SchemaHash:    registration.SchemaHash,
			TimeoutMs:     registration.TimeoutMs,
			Concurrency:   registration.Concurrency,
			NoRetry:       registration.NoRetry,
		})
	}
	return vslice.SortBy(items, func(a skeled.EventListenerRegistration, b skeled.EventListenerRegistration) bool {
		return cmp.Compare(a.EventSkelName, b.EventSkelName) < 0
	})
}

func toSkeledTaskRunnerRegistrations(registrations []core.TaskRunnerRegistration) []skeled.TaskRunnerRegistration {
	items := []skeled.TaskRunnerRegistration{}
	for _, registration := range registrations {
		items = append(items, skeled.TaskRunnerRegistration{
			TaskSkelName:   registration.TaskSkelName,
			SchemaHash:     registration.SchemaHash,
			TimeoutMs:      registration.TimeoutMs,
			Concurrency:    registration.Concurrency,
			NoRetry:        registration.NoRetry,
			CronSchedulers: toSkeledTaskRunnerCronSchedulers(registration.CronSchedulers),
		})
	}
	return vslice.SortBy(items, func(a skeled.TaskRunnerRegistration, b skeled.TaskRunnerRegistration) bool {
		return cmp.Compare(a.TaskSkelName, b.TaskSkelName) < 0
	})
}

func toSkeledTaskRunnerCronSchedulers(schedulers []core.TaskRunnerCronScheduler) []skeled.TaskRunnerCronScheduler {
	items := []skeled.TaskRunnerCronScheduler{}
	for _, scheduler := range schedulers {
		items = append(items, skeled.TaskRunnerCronScheduler{
			TriggerSkelName: scheduler.TriggerSkelName,
			CronExpr:        scheduler.CronExpr,
		})
	}
	return items
}
