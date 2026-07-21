package minder

import (
	"context"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type Mutator interface {
	OnSetup(instance *AppInstance)
	OnDrain(instance *AppInstance)
	OnDestroy(instance *AppInstance)
}

const (
	unregisterPropagationGrace = 400 * time.Millisecond
	unregisterDrainTimeout     = 30 * time.Second
)

type AppMinder struct {
	app.BaseModule

	Context               context.Context                 `inject:""`
	Flag                  *flag.Flag                      `inject:""`
	App                   runtime.App                     `inject:""`
	InprocFlag            *app.InternalInprocFlag         `inject:""`
	RegistryServiceClient hubskeled.RegistryServiceClient `inject:""`

	mutex        sync.Mutex
	instanceByID map[string]*AppInstance
	mutators     []Mutator
}

func (m *AppMinder) DIInit() {
	m.instanceByID = map[string]*AppInstance{}
}

func (m *AppMinder) AfterAppStop() {
	for _, instance := range m.appInstances() {
		instance.stopHealthcheck()
		instance.stopHeartbeat()
	}
}

type AppRegistration struct {
	AppInfo           runtime.App
	ConsoleEndpoint   string
	ServiceEndpoint   string
	WebEndpointPrefix string
	EventEndpoint     string
	TaskEndpoint      string
	IngressEndpoint   string
	ServiceHandlers   []skeled.ServiceHandlerRegistration
	WebHandlers       []skeled.WebHandlerRegistration
	EventListeners    []skeled.EventListenerRegistration
	TaskRunners       []skeled.TaskRunnerRegistration
	DomainSchemas     []skel.JSON
}

func (m *AppMinder) RegisterInstance(registration AppRegistration) {
	instance := m.newAppInstance(registration)
	if !m.addInstance(instance) {
		return
	}

	instance.startHeartbeat()
	instance.startHealthcheck(func() {
		m.unregisterInstance(instance)
	})

	m.beforeRegistration(instance)
	instance.registerHubInstance()
}

func (m *AppMinder) UnregisterInstance(instanceID string) {
	if instance, ok := m.appInstance(instanceID); ok {
		m.unregisterInstance(instance)
	}
}

func (m *AppMinder) unregisterInstance(instance *AppInstance) {
	instance.unregisterHubInstance()
	instance.stopHealthcheck()
	instance.stopHeartbeat()
	time.Sleep(unregisterPropagationGrace)

	instance.BeginDrain()
	m.onDrain(instance)

	drainCtx, cancel := context.WithTimeout(m.Context, unregisterDrainTimeout)
	defer cancel()
	instance.WaitDrain(drainCtx)

	m.onDestroy(instance)
	m.removeInstance(instance)
}
