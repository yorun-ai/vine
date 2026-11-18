package event

import (
	"context"
	"sync"
	"uuid"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

type Manager struct {
	app.BaseModule

	Context    context.Context   `inject:""`
	CurrentApp meta.CurrentApp   `inject:""`
	NATSClient *linknats.Client  `inject:""`
	AppMinder  *minder.AppMinder `inject:""`

	mutex                    sync.Mutex
	listenersByAppInstanceID map[uuid.UUID][]*_EventListenerState
}

type _EventListenerState struct {
	instance       *minder.AppInstance
	appInstanceID  uuid.UUID
	eventEndpoint  string
	registration   skeled.EventListenerRegistration
	semaphore      chan struct{}
	consumeContext linknats.ConsumeContext
}

func (m *Manager) DIInit() {
	m.listenersByAppInstanceID = map[uuid.UUID][]*_EventListenerState{}
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
	m.listenersByAppInstanceID = map[uuid.UUID][]*_EventListenerState{}
	m.mutex.Unlock()

	for _, consumeContext := range consumeContexts {
		consumeContext.Stop()
	}
}
