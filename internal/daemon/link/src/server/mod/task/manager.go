package task

import (
	"context"
	"sync"

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

	mutex                      sync.Mutex
	runnerByAppInstanceID      map[string]map[string]*_TaskRunnerState
	subscriptionByTaskSkelName map[string]*_TaskSubscription
}

type _TaskRunnerState struct {
	instance      *minder.AppInstance
	appInstanceID string
	taskEndpoint  string
	registration  skeled.TaskRunnerRegistration
	semaphore     chan struct{}
	readContext   context.Context
	cancelRead    context.CancelFunc
	subscription  *_TaskSubscription
}

type _TaskSubscription struct {
	messages    linknats.MessagesContext
	runnerByApp map[string]*_TaskRunnerState
	nextRunner  int
	changed     chan struct{}
	stopped     bool
	done        chan struct{}
}

func (m *Manager) DIInit() {
	m.runnerByAppInstanceID = map[string]map[string]*_TaskRunnerState{}
	m.subscriptionByTaskSkelName = map[string]*_TaskSubscription{}
	m.AppMinder.AddMutator(m)
}

func (m *Manager) AfterAppStop() {
	m.mutex.Lock()
	states := make([]*_TaskSubscription, 0, len(m.subscriptionByTaskSkelName))
	for _, state := range m.subscriptionByTaskSkelName {
		for _, runner := range state.runnerByApp {
			runner.cancelRead()
		}
		m.stopSubscriptionLocked(state)
		states = append(states, state)
	}
	m.runnerByAppInstanceID = map[string]map[string]*_TaskRunnerState{}
	m.subscriptionByTaskSkelName = map[string]*_TaskSubscription{}
	m.mutex.Unlock()
	for _, state := range states {
		<-state.done
	}
}

func (m *Manager) stopSubscriptionLocked(state *_TaskSubscription) {
	state.stopped = true
	m.notifyCapacityLocked(state)
	state.messages.Stop()
}

func (m *Manager) notifyCapacityLocked(state *_TaskSubscription) {
	close(state.changed)
	state.changed = make(chan struct{})
}
