package minder

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/util/vslice"
)

type AppInstance struct {
	minder *AppMinder
	mutex  sync.Mutex

	AppInfo            runtime.App
	ConsoleEndpoint    string
	ServiceEndpoint    string
	WebEndpointPrefix  string
	EventEndpoint      string
	TaskEndpoint       string
	IngressEndpoint    string
	ServiceHandlers    []skeled.ServiceHandlerRegistration
	WebHandlers        []skeled.WebHandlerRegistration
	EventListeners     []skeled.EventListenerRegistration
	TaskRunners        []skeled.TaskRunnerRegistration
	DomainSchemas      []skel.JSON
	HubServiceHandlers []hubskeled.ServiceHandlerRegistration
	HubWebHandlers     []hubskeled.WebHandlerRegistration

	heartbeatCancel   context.CancelFunc
	healthcheckCancel context.CancelFunc
	draining          bool
	inflight          int
	idleCh            chan struct{}
}

func (m *AppMinder) newAppInstance(registration AppRegistration) *AppInstance {
	serviceHandlers := vslice.Clone(registration.ServiceHandlers)
	webHandlers := vslice.Clone(registration.WebHandlers)
	return &AppInstance{
		minder:             m,
		AppInfo:            registration.AppInfo,
		ConsoleEndpoint:    registration.ConsoleEndpoint,
		ServiceEndpoint:    registration.ServiceEndpoint,
		WebEndpointPrefix:  registration.WebEndpointPrefix,
		EventEndpoint:      registration.EventEndpoint,
		TaskEndpoint:       registration.TaskEndpoint,
		IngressEndpoint:    registration.IngressEndpoint,
		ServiceHandlers:    serviceHandlers,
		WebHandlers:        webHandlers,
		EventListeners:     vslice.Clone(registration.EventListeners),
		TaskRunners:        vslice.Clone(registration.TaskRunners),
		DomainSchemas:      vslice.Clone(registration.DomainSchemas),
		HubServiceHandlers: defaultHubServiceHandlerRegistrations(serviceHandlers),
		HubWebHandlers:     defaultHubWebHandlerRegistrations(webHandlers),
		idleCh:             newClosedSignalChan(),
	}
}

func newClosedSignalChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (i *AppInstance) TryStartWork() bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.draining {
		return false
	}
	i.inflight++
	if i.inflight == 1 {
		i.idleCh = make(chan struct{})
	}
	return true
}

func (i *AppInstance) FinishWork() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.inflight == 0 {
		return
	}
	i.inflight--
	if i.inflight == 0 {
		close(i.idleCh)
	}
}

func (i *AppInstance) BeginDrain() {
	i.mutex.Lock()
	i.draining = true
	i.mutex.Unlock()
}

func (i *AppInstance) WaitDrain(ctx context.Context) bool {
	i.mutex.Lock()
	idleCh := i.idleCh
	i.mutex.Unlock()

	select {
	case <-idleCh:
		return true
	case <-ctx.Done():
		return false
	}
}

func defaultHubServiceHandlerRegistrations(registrations []skeled.ServiceHandlerRegistration) []hubskeled.ServiceHandlerRegistration {
	ret := make([]hubskeled.ServiceHandlerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.ServiceHandlerRegistration{
			ServiceSkelName: registration.ServiceSkelName,
			SchemaHash:      registration.SchemaHash,
		})
	}
	return ret
}

func defaultHubWebHandlerRegistrations(registrations []skeled.WebHandlerRegistration) []hubskeled.WebHandlerRegistration {
	ret := make([]hubskeled.WebHandlerRegistration, 0, len(registrations))
	for _, registration := range registrations {
		ret = append(ret, hubskeled.WebHandlerRegistration{
			WebSkelName: registration.WebSkelName,
			SchemaHash:  registration.SchemaHash,
		})
	}
	return ret
}
