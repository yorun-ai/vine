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
	instance      *minder.AppInstance
	appInstanceID uuid.UUID
	eventEndpoint string
	registration  skeled.EventListenerRegistration
	semaphore     chan struct{}
	cancelRead    context.CancelFunc
	done          chan struct{}
}

func (m *Manager) DIInit() {
	m.listenersByAppInstanceID = map[uuid.UUID][]*_EventListenerState{}
	m.AppMinder.AddMutator(m)
}

func (m *Manager) AfterAppStop() {
	m.mutex.Lock()
	listeners := m.listenersByAppInstanceID
	m.listenersByAppInstanceID = nil
	m.mutex.Unlock()

	for _, group := range listeners {
		stopEventListeners(group)
	}
}

func stopEventListeners(listeners []*_EventListenerState) {
	for _, listener := range listeners {
		listener.cancelRead()
	}
	for _, listener := range listeners {
		<-listener.done
	}
}
