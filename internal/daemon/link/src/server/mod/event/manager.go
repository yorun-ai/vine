package event

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/runtime"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

type Manager struct {
	app.BaseModule

	Context    context.Context   `inject:""`
	App        runtime.App       `inject:""`
	NATSClient *linknats.Client  `inject:""`
	AppMinder  *minder.AppMinder `inject:""`

	mutex                    sync.Mutex
	listenersByAppInstanceID map[string][]*_EventListenerState
}

type _EventListenerState struct {
	instance       *minder.AppInstance
	appInstanceID  string
	eventEndpoint  string
	registration   skeled.EventListenerRegistration
	semaphore      chan struct{}
	consumeContext linknats.ConsumeContext
}

func (m *Manager) DIInit() {
	m.listenersByAppInstanceID = map[string][]*_EventListenerState{}
	m.AppMinder.AddMutator(m)
}

func (m *Manager) AfterAppStop() {
	m.mutex.Lock()
	consumeContexts := []linknats.ConsumeContext(nil)
	for _, listeners := range m.listenersByAppInstanceID {
		for _, listenerState := range listeners {
			consumeContexts = append(consumeContexts, listenerState.consumeContext)
		}
	}
	m.listenersByAppInstanceID = map[string][]*_EventListenerState{}
	m.mutex.Unlock()

	for _, consumeContext := range consumeContexts {
		consumeContext.Stop()
	}
}
