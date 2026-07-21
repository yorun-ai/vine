package task

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
}

type _TaskSubscription struct {
	consumeContext linknats.ConsumeContext
	runnerByApp    map[string]*_TaskRunnerState
	nextRunner     int
}

func (m *Manager) DIInit() {
	m.runnerByAppInstanceID = map[string]map[string]*_TaskRunnerState{}
	m.subscriptionByTaskSkelName = map[string]*_TaskSubscription{}
	m.AppMinder.AddMutator(m)
}

func (m *Manager) AfterAppStop() {
	m.mutex.Lock()
	consumeContexts := make([]linknats.ConsumeContext, 0, len(m.subscriptionByTaskSkelName))
	for _, state := range m.subscriptionByTaskSkelName {
		consumeContexts = append(consumeContexts, state.consumeContext)
	}
	m.runnerByAppInstanceID = map[string]map[string]*_TaskRunnerState{}
	m.subscriptionByTaskSkelName = map[string]*_TaskSubscription{}
	m.mutex.Unlock()

	for _, consumeContext := range consumeContexts {
		consumeContext.Stop()
	}
}
