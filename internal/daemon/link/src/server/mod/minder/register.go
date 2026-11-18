package minder

import (
	"time"
	"uuid"

	"go.yorun.ai/vine/internal/core/link/skeled"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/util/vslice"
)

// Bound in-flight Hub calls so they leave time for unregister and local drain
// within the application's shutdown timeout.
const hubRegistryTimeout = 5 * time.Second

func (i *AppInstance) registerHubInstance() {
	i.hubMutex.Lock()
	defer i.hubMutex.Unlock()
	i.registerHubInstanceLocked()
}

func (i *AppInstance) registerHubInstanceLocked() {
	if i.isStopping() {
		return
	}
	i.minder.RegistryServiceClient.Register(hubskeled.AppRegistration{
		Name:              i.AppInfo.Name(),
		InstanceId:        uuid.MustParse(i.AppInfo.InstanceId()),
		Version:           i.AppInfo.Version(),
		Endpoint:          i.IngressEndpoint,
		ServiceHandlers:   vslice.Clone(i.HubServiceHandlers),
		WebHandlers:       vslice.Clone(i.HubWebHandlers),
		EventListeners:    toHubEventListenerRegistrations(i.EventListeners),
		TaskRunners:       toHubTaskRunnerRegistrations(i.TaskRunners),
		DomainDescriptors: vslice.Clone(i.DomainDescriptors),
	}, rpcclient.WithTimeout(hubRegistryTimeout))
}

func (i *AppInstance) unregisterHubInstance() {
	i.hubMutex.Lock()
	defer i.hubMutex.Unlock()
	i.minder.RegistryServiceClient.Unregister(i.AppInfo.Name(), uuid.MustParse(i.AppInfo.InstanceId()), rpcclient.WithTimeout(hubRegistryTimeout))
}

func toHubEventListenerRegistrations(registrations []skeled.EventListenerRegistration) []hubskeled.EventListenerRegistration {
	ret := make([]hubskeled.EventListenerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.EventListenerRegistration{
			EventSkelName:  registration.EventSkelName,
			DescriptorHash: registration.DescriptorHash,
			TimeoutMs:      registration.TimeoutMs,
			Concurrency:    registration.Concurrency,
			NoRetry:        registration.NoRetry,
		})
	}
	return ret
}

func toHubTaskRunnerRegistrations(registrations []skeled.TaskRunnerRegistration) []hubskeled.TaskRunnerRegistration {
	ret := make([]hubskeled.TaskRunnerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.TaskRunnerRegistration{
			TaskSkelName:   registration.TaskSkelName,
			DescriptorHash: registration.DescriptorHash,
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
