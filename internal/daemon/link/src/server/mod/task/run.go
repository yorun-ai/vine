package task

import (
	"context"
	"errors"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/runtime"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

var newAppTaskServiceClient = func(ctx context.Context, clientApp runtime.App, endpoint string, msg taskspec.NATSMessage) appskeled.TaskServiceClientER {
	trace, err := meta.NewTrace(msg.Metadata.TraceId, msg.Metadata.TraceSpan)
	ex.PanicIfError(err)
	actor := meta.NewAbsentActor()
	rpcCtx := meta.NewContext(ctx, trace, nil, actor)
	return appskeled.NewTaskServiceClientER(rpcclient.New(rpcclient.Option{
		Context:             rpcCtx,
		ClientApp:           clientApp,
		Logger:              logger.NewLogger(logger.GlobalOption()),
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
		runnerByTaskSkel[taskSkelName] = &_TaskRunnerState{
			instance:      instance,
			appInstanceID: instance.AppInfo.InstanceId(),
			taskEndpoint:  instance.TaskEndpoint,
			registration:  registration,
			semaphore:     make(chan struct{}, registration.Concurrency),
		}

		subscriptionState, exists := m.subscriptionByTaskSkelName[taskSkelName]
		if !exists {
			subscriptionState = &_TaskSubscription{
				consumeContext: m.NATSClient.Consume(
					taskStreamConfig(),
					taskspec.NATSSubject(taskSkelName),
					taskspec.NATSConsumerName(taskSkelName),
					m.newTaskMessageHandler(taskSkelName),
				),
				runnerByApp: map[string]*_TaskRunnerState{},
			}
			m.subscriptionByTaskSkelName[taskSkelName] = subscriptionState
		}
		subscriptionState.runnerByApp[instance.AppInfo.InstanceId()] = runnerByTaskSkel[taskSkelName]
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
	for taskSkelName := range runnerByTaskSkel {
		subscriptionState := m.subscriptionByTaskSkelName[taskSkelName]
		delete(subscriptionState.runnerByApp, instanceId)
		if len(subscriptionState.runnerByApp) > 0 {
			continue
		}

		delete(m.subscriptionByTaskSkelName, taskSkelName)
		subscriptionState.consumeContext.Stop()
	}
}

func (m *Manager) newTaskMessageHandler(taskSkelName string) func(msg jetstream.Msg) {
	return func(natsMsg jetstream.Msg) {
		msg := *vcode.MustUnmarshalJson[*taskspec.NATSMessage](natsMsg.Data())

		m.mutex.Lock()
		subscriptionState, exists := m.subscriptionByTaskSkelName[taskSkelName]
		if !exists {
			m.mutex.Unlock()
			ackTaskMessage(natsMsg.Term())
			return
		}

		runnerAppInstanceIDs := make([]string, 0, len(subscriptionState.runnerByApp))
		for appInstanceID := range subscriptionState.runnerByApp {
			runnerAppInstanceIDs = append(runnerAppInstanceIDs, appInstanceID)
		}
		runnerAppInstanceIDs = vslice.Sort(runnerAppInstanceIDs)
		runner := subscriptionState.runnerByApp[runnerAppInstanceIDs[subscriptionState.nextRunner%len(runnerAppInstanceIDs)]]
		subscriptionState.nextRunner++
		m.mutex.Unlock()
		if !runner.instance.TryStartWork() {
			if runner.registration.NoRetry {
				ackTaskMessage(natsMsg.Term())
			} else {
				ackTaskMessage(natsMsg.Nak())
			}
			return
		}

		go m.runTaskMessage(natsMsg, runner, msg)
	}
}

func (m *Manager) runTaskMessage(natsMsg jetstream.Msg, runner *_TaskRunnerState, msg taskspec.NATSMessage) {
	if runner == nil {
		ackTaskMessage(natsMsg.Term())
		return
	}
	defer runner.instance.FinishWork()

	err := m.runTask(runner, msg)
	if err != nil && !runner.registration.NoRetry {
		ackTaskMessage(natsMsg.Nak())
		return
	}

	ackTaskMessage(natsMsg.Ack())
}

func (m *Manager) runTask(runner *_TaskRunnerState, msg taskspec.NATSMessage) ex.Error {
	runner.semaphore <- struct{}{}
	defer func() {
		<-runner.semaphore
	}()

	client := newAppTaskServiceClient(m.Context, m.App, runner.taskEndpoint, msg)
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
		logger.Error("task app task call failed",
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
