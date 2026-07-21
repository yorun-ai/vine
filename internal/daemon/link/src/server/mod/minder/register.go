package minder

import (
	"github.com/google/uuid"

	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/util/vslice"
)

func (i *AppInstance) registerHubInstance() {
	i.minder.RegistryServiceClient.Register(hubskeled.AppRegistration{
		Name:            i.AppInfo.Name(),
		InstanceId:      skel.NewUUID(uuid.MustParse(i.AppInfo.InstanceId())),
		Version:         i.AppInfo.Version(),
		Endpoint:        i.IngressEndpoint,
		ServiceHandlers: vslice.Clone(i.HubServiceHandlers),
		WebHandlers:     vslice.Clone(i.HubWebHandlers),
		EventListeners:  toHubEventListenerRegistrations(i.EventListeners),
		TaskRunners:     toHubTaskRunnerRegistrations(i.TaskRunners),
		DomainSchemas:   vslice.Clone(i.DomainSchemas),
	})
}

func (i *AppInstance) unregisterHubInstance() {
	i.minder.RegistryServiceClient.Unregister(i.AppInfo.Name(), skel.NewUUID(uuid.MustParse(i.AppInfo.InstanceId())))
}

func toHubEventListenerRegistrations(registrations []skeled.EventListenerRegistration) []hubskeled.EventListenerRegistration {
	ret := make([]hubskeled.EventListenerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.EventListenerRegistration{
			EventSkelName: registration.EventSkelName,
			SchemaHash:    registration.SchemaHash,
			TimeoutMs:     registration.TimeoutMs,
			Concurrency:   registration.Concurrency,
			NoRetry:       registration.NoRetry,
		})
	}
	return ret
}

func toHubTaskRunnerRegistrations(registrations []skeled.TaskRunnerRegistration) []hubskeled.TaskRunnerRegistration {
	ret := make([]hubskeled.TaskRunnerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.TaskRunnerRegistration{
			TaskSkelName:   registration.TaskSkelName,
			SchemaHash:     registration.SchemaHash,
			TimeoutMs:      registration.TimeoutMs,
			Concurrency:    registration.Concurrency,
			NoRetry:        registration.NoRetry,
			CronSchedulers: toHubTaskRunnerCronSchedulers(registration.CronSchedulers),
		})
	}
	return ret
}

func toHubTaskRunnerCronSchedulers(schedulers []skeled.TaskRunnerCronScheduler) []hubskeled.TaskRunnerCronScheduler {
	ret := make([]hubskeled.TaskRunnerCronScheduler, 0, len(schedulers))
	for _, scheduler := range schedulers {
		ret = append(ret, hubskeled.TaskRunnerCronScheduler{
			TriggerSkelName: scheduler.TriggerSkelName,
			CronExpr:        scheduler.CronExpr,
		})
	}
	return ret
}
