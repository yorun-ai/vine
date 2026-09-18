package task

import (
	"context"
	"errors"
	"sync"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

var taskLogger = logger.New("daemon:link:task")

var newAppTaskServiceClient = func(ctx context.Context, clientApp meta.App, endpoint string, msg taskspec.NATSMessage) appskeled.TaskServiceClientER {
	trace, err := meta.NewTrace(msg.Metadata.TraceId, msg.Metadata.TraceSpan)
	ex.PanicIfError(err)
	actor := meta.NewAbsentActor()
	rpcCtx := meta.NewContext(ctx, trace, nil, actor)
	return appskeled.NewTaskServiceClientER(rpcclient.New(rpcclient.Option{
		Context:             rpcCtx,
		ClientApp:           clientApp,
		Logger:              taskLogger.Child("client"),
		ReturnIfSystemError: true,
		ServerEndpoint:      endpoint,
	}))
}

var runAppTask = func(client appskeled.TaskServiceClientER, run appskeled.TaskRun, timeout time.Duration) ex.Error {
	return client.RunTask(run, rpcclient.WithTimeout(timeout))
}

func (m *Manager) OnSetup(instance *minder.AppInstance) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(instance.TaskRunners) == 0 {
		return
	}

	runnerByTaskSkel := map[string]*_TaskRunnerState{}
	m.runnerByAppInstanceID[instance.AppInfo.InstanceId()] = runnerByTaskSkel
	for _, registration := range instance.TaskRunners {
		taskSkelName := registration.TaskSkelName
		readContext, cancelRead := context.WithCancel(m.Context)
		runnerByTaskSkel[taskSkelName] = new(_TaskRunnerState{
			readContext:   readContext,
			cancelRead:    cancelRead,
			instance:      instance,
			appInstanceID: instance.AppInfo.InstanceId(),
			taskEndpoint:  instance.TaskEndpoint,
			registration:  registration,
			semaphore:     make(chan struct{}, registration.Concurrency),
		})

		subscriptionState, exists := m.subscriptionByTaskSkelName[taskSkelName]
		if !exists {
			subscriptionState = new(_TaskSubscription{
				messages: m.NATSClient.Messages(
					m.Context,
					taskStreamConfig(),
					taskspec.NATSSubject(taskSkelName),
					taskspec.NATSConsumerName(taskSkelName),
				),
				runnerByApp: map[string]*_TaskRunnerState{},
				changed:     make(chan struct{}),
				done:        make(chan struct{}),
			})
			m.subscriptionByTaskSkelName[taskSkelName] = subscriptionState
			go m.dispatchTaskMessages(subscriptionState)
		}
		subscriptionState.runnerByApp[instance.AppInfo.InstanceId()] = runnerByTaskSkel[taskSkelName]
		runnerByTaskSkel[taskSkelName].subscription = subscriptionState
		m.notifyCapacityLocked(subscriptionState)
	}
}

func (m *Manager) OnDrain(instance *minder.AppInstance) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.removeAppLocked(instance.AppInfo.InstanceId())
}

func (*Manager) OnDestroy(*minder.AppInstance) {}

func (m *Manager) removeAppLocked(instanceId string) {
	runnerByTaskSkel, exists := m.runnerByAppInstanceID[instanceId]
	if !exists {
		return
	}

	delete(m.runnerByAppInstanceID, instanceId)
	for taskSkelName, runner := range runnerByTaskSkel {
		runner.cancelRead()
		subscriptionState := m.subscriptionByTaskSkelName[taskSkelName]
		delete(subscriptionState.runnerByApp, instanceId)
		m.notifyCapacityLocked(subscriptionState)
		if len(subscriptionState.runnerByApp) > 0 {
			continue
		}

		delete(m.subscriptionByTaskSkelName, taskSkelName)
		m.stopSubscriptionLocked(subscriptionState)
	}
}

// Each task subscription has one dispatcher. It reserves capacity before Next,
// so neither decoded messages nor goroutines form an unbounded local queue.
func (m *Manager) dispatchTaskMessages(state *_TaskSubscription) {
	defer close(state.done)
	defer state.messages.Stop()
	for {
		runner := m.nextAvailableRunner(state)
		if runner == nil {
			return
		}
		natsMsg, err := state.messages.Next(runner.readContext)
		if err != nil {
			m.releaseTaskSlot(runner)
			if m.Context.Err() != nil || errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				return
			}
			if runner.readContext.Err() != nil {
				continue
			}
			taskLogger.Warn("task message read failed", "taskSkelName", runner.registration.TaskSkelName, "error", err)
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-runner.readContext.Done():
			case <-timer.C:
			}
			timer.Stop()
			continue
		}
		// A reservation is not inflight work: idle reads must not delay app drain.
		// Recheck admission after Next, since the selected runner may have drained.
		m.mutex.Lock()
		admitted := !state.stopped && runner.readContext.Err() == nil && runner.instance.TryStartWork()
		m.mutex.Unlock()
		if !admitted {
			ackTaskMessage(natsMsg.Nak())
			m.releaseTaskSlot(runner)
			continue
		}
		go m.runTaskMessage(natsMsg, runner)
	}
}

func (m *Manager) nextAvailableRunner(state *_TaskSubscription) *_TaskRunnerState {
	for {
		m.mutex.Lock()
		if state.stopped || m.Context.Err() != nil {
			m.mutex.Unlock()
			return nil
		}
		ids := make([]string, 0, len(state.runnerByApp))
		for id := range state.runnerByApp {
			ids = append(ids, id)
		}
		ids = vslice.Sort(ids)
		for offset := 0; offset < len(ids); offset++ {
			runner := state.runnerByApp[ids[(state.nextRunner+offset)%len(ids)]]
			select {
			case runner.semaphore <- struct{}{}:
				state.nextRunner = (state.nextRunner + offset + 1) % len(ids)
				m.mutex.Unlock()
				return runner
			default:
			}
		}
		changed := state.changed
		m.mutex.Unlock()
		select {
		case <-m.Context.Done():
			return nil
		case <-changed:
		}
	}
}

func (m *Manager) releaseTaskSlot(runner *_TaskRunnerState) {
	m.mutex.Lock()
	<-runner.semaphore
	m.notifyCapacityLocked(runner.subscription)
	m.mutex.Unlock()
}

func (m *Manager) runTaskMessage(natsMsg jetstream.Msg, runner *_TaskRunnerState) {
	defer m.releaseTaskSlot(runner)
	defer runner.instance.FinishWork()
	// Renew only admitted work. The renewal goroutine is bounded by the same
	// execution slots and is joined before the final acknowledgement.
	stopRenewal := keepTaskMessageAlive(natsMsg)
	defer stopRenewal()
	msg := *vcode.MustUnmarshalJson[*taskspec.NATSMessage](natsMsg.Data())

	err := m.runTask(runner, msg)
	stopRenewal()
	if err != nil && !runner.registration.NoRetry {
		ackTaskMessage(natsMsg.Nak())
		return
	}

	ackTaskMessage(natsMsg.Ack())
}

func keepTaskMessageAlive(msg jetstream.Msg) func() {
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(linknats.MessageAckWait / 3)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := msg.InProgress(); err != nil {
					taskLogger.Warn("task acknowledgement renewal failed", "error", err)
				}
			}
		}
	}()
	return sync.OnceFunc(func() { close(stop); <-done })
}

func (m *Manager) runTask(runner *_TaskRunnerState, msg taskspec.NATSMessage) ex.Error {
	client := newAppTaskServiceClient(m.Context, m.CurrentApp, runner.taskEndpoint, msg)
	run := appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       msg.Metadata.TraceId,
			TraceSpan:     msg.Metadata.TraceSpan,
			AppName:       msg.Metadata.AppName,
			AppVersion:    msg.Metadata.AppVersion,
			AppInstanceId: msg.Metadata.AppInstanceId,
			LaunchedAt:    msg.Metadata.LaunchedAt,
		},
		TaskSkelName:    msg.TaskSkelName,
		TriggerSkelName: msg.TriggerSkelName,
		ArgumentsJson:   msg.ArgumentsJson,
	}

	err := runAppTask(client, run, time.Duration(runner.registration.TimeoutMs)*time.Millisecond)
	if err != nil {
		taskLogger.Error("task app task call failed",
			"taskSkelName", msg.TaskSkelName,
			"triggerSkelName", msg.TriggerSkelName,
			"endpoint", runner.taskEndpoint,
			"instanceId", runner.appInstanceID,
			"error", err,
		)
	}
	return err
}

func ackTaskMessage(err error) {
	if err == nil || errors.Is(err, nats.ErrConnectionClosed) || errors.Is(err, nats.ErrConnectionDraining) {
		return
	}
	ex.PanicIfError(err)
}
