package minder

import (
	"context"
	"sync"
	"time"
	"uuid"

	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
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
	CurrentApp            meta.CurrentApp                 `inject:""`
	InprocFlag            *app.InternalInprocFlag         `inject:""`
	HubInfo               *hubinfo.HubInfo                `inject:""`
	RegistryServiceClient hubskeled.RegistryServiceClient `inject:""`

	mutex        sync.Mutex
	instanceByID map[uuid.UUID]*AppInstance
	mutators     []Mutator
}

func (m *AppMinder) DIInit() {
	m.instanceByID = map[uuid.UUID]*AppInstance{}
}

func (m *AppMinder) AfterAppStop() {
	for _, instance := range m.appInstances() {
		instance.stopRegistration()
		instance.hubMutex.Lock()
		instance.hubMutex.Unlock()
	}
}

type AppRegistration struct {
	AppInfo           meta.App
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
	DomainDescriptors []skeltype.JSON
}

func (m *AppMinder) RegisterInstance(registration AppRegistration) {
	instance := m.newAppInstance(registration)
	instance.hubMutex.Lock()
	defer instance.hubMutex.Unlock()
	if !m.addInstance(instance) {
		return
	}

	instance.startHeartbeat()
	instance.startHealthcheck(func() {
		m.unregisterInstance(instance)
	})

	m.beforeRegistration(instance)
	instance.registerHubInstanceLocked()
}

func (m *AppMinder) UnregisterInstance(instanceID uuid.UUID) {
	if instance, ok := m.appInstance(instanceID); ok {
		m.unregisterInstance(instance)
	}
}

func (m *AppMinder) unregisterInstance(instance *AppInstance) {
	done, first := instance.beginUnregister()
	if !first {
		<-done
		return
	}
	defer close(done)
	instance.stopRegistration()
	// Hub failure must still stop local delivery and release instance ownership.
	// Preserve the failure for the caller after cleanup has completed.
	defer m.drainInstance(instance)
	instance.unregisterHubInstance()
}

func (m *AppMinder) drainInstance(instance *AppInstance) {
	defer m.removeInstance(instance)
	time.Sleep(unregisterPropagationGrace)

	instance.BeginDrain()
	m.onDrain(instance)

	drainCtx, cancel := context.WithTimeout(m.Context, unregisterDrainTimeout)
	defer cancel()
	instance.WaitDrain(drainCtx)

	m.onDestroy(instance)
}
