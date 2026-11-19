package minder

import (
	"context"
	"sync"
	"time"
	"uuid"

	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/internal/util/goutil"
	"go.yorun.ai/vine/util/vpre"
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
		instance.initialRegistration.Wait()
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
	instance.initialRegistration.Add(1)
	defer instance.initialRegistration.Done()
	if existing := m.addInstance(instance); existing != instance {
		existing.initialRegistration.Wait()
		if existing.registrationFailure != nil {
			panic(existing.registrationFailure)
		}
		vpre.Check(!existing.isStopping(), "instance %s is stopping", existing.AppInfo.InstanceId())
		return
	}
	instance.hubMutex.Lock()
	defer instance.hubMutex.Unlock()

	mutators := m.mutatorSnapshot()
	started := 0
	hubAttempted := false
	defer func() {
		if failure := recover(); failure != nil {
			instance.registrationFailure = failure
			instance.stopRegistration()
			instance.BeginDrain()
			if hubAttempted {
				// Hub may have applied the registration before its reply was lost.
				goutil.RunWithRecover(func(err any) {
					logger.Warn("minder could not roll back Hub registration", "instanceId", instance.AppInfo.InstanceId(), "error", err)
				}, instance.unregisterHubInstanceLocked)
			}
			m.drainInstance(instance, mutators[:started])
			panic(failure)
		}
	}()

	instance.startHeartbeat()
	instance.startHealthcheck(func() {
		m.unregisterInstance(instance)
	})

	for index, mutator := range mutators {
		started = index + 1 // OnSetup can fail after acquiring resources.
		mutator.OnSetup(instance)
	}
	hubAttempted = true
	instance.registerHubInstanceLocked()
}

func (m *AppMinder) UnregisterInstance(instanceID uuid.UUID) {
	if instance, ok := m.appInstance(instanceID); ok {
		m.unregisterInstance(instance)
	}
}

func (m *AppMinder) unregisterInstance(instance *AppInstance) {
	instance.initialRegistration.Wait()
	if instance.registrationFailure != nil {
		return
	}
	done, first := instance.beginUnregister()
	if !first {
		<-done
		return
	}
	defer close(done)
	instance.stopRegistration()
	// Hub failure must still stop local delivery and release instance ownership.
	// Preserve the failure for the caller after cleanup has completed.
	defer func() {
		time.Sleep(unregisterPropagationGrace)
		m.drainInstance(instance, m.mutatorSnapshot())
	}()
	instance.unregisterHubInstance()
}

func (m *AppMinder) drainInstance(instance *AppInstance, mutators []Mutator) {
	defer m.removeInstance(instance)

	instance.BeginDrain()
	for _, mutator := range mutators {
		mutator.OnDrain(instance)
	}

	drainCtx, cancel := context.WithTimeout(m.Context, unregisterDrainTimeout)
	defer cancel()
	instance.WaitDrain(drainCtx)

	for _, mutator := range mutators {
		mutator.OnDestroy(instance)
	}
}
