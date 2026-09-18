package task

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go/jetstream"
	"go.yorun.ai/vine/util/vcode"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const testPathTask = "/task"

func TestManagerRegistersListenerAndDispatchesRun(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppTaskServiceClient
	oldRun := runAppTask
	defer func() {
		newAppTaskServiceClient = oldFactory
		runAppTask = oldRun
	}()

	hooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppTaskServiceClient = func(context.Context, meta.App, string, taskspec.NATSMessage) appskeled.TaskServiceClientER {
		return &_ManagerAppTaskClient{
			runTask: func(run appskeled.TaskRun) error {
				hooks.mutex.Lock()
				hooks.runs = append(hooks.runs, run)
				hooks.mutex.Unlock()
				hooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppTask = func(client appskeled.TaskServiceClientER, run appskeled.TaskRun, timeout time.Duration) ex.Error {
		hooks.mutex.Lock()
		hooks.timeout = timeout
		hooks.callCount++
		hooks.mutex.Unlock()
		return client.RunTask(run)
	}

	appInfo, err := meta.NewApp("demo.app", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	endpoint := testLocalAppEndpoint(8080)
	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:      appInfo,
		TaskEndpoint: endpoint + testPathTask,
		TaskRunners: []skeled.TaskRunnerRegistration{{
			TaskSkelName: "demo.user.SyncUserTask",
			TimeoutMs:    2500,
			Concurrency:  1,
		}},
	})

	manager.LaunchTask(skeled.TaskLaunch{
		Metadata: skeled.TaskLaunchMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		TaskSkelName:    "demo.user.SyncUserTask",
		TriggerSkelName: "demo.user.SyncUserTaskManualTrigger",
		ArgumentsJson:   `{"userId":"u1"}`,
	})

	requireDispatchSignal(t, hooks.completed, "task dispatch timeout")

	hooks.mutex.Lock()
	defer hooks.mutex.Unlock()
	assert.Equal(t, time.Duration(2500)*time.Millisecond, hooks.timeout)
	assert.Equal(t, 1, hooks.callCount)
	assert.Equal(t, "demo.user.SyncUserTask", hooks.runs[0].TaskSkelName)
	assert.Equal(t, "demo.user.SyncUserTaskManualTrigger", hooks.runs[0].TriggerSkelName)
	assert.Equal(t, `{"userId":"u1"}`, hooks.runs[0].ArgumentsJson)
	assert.Equal(t, "launcher.app", hooks.runs[0].Metadata.AppName)
}

func TestManagerLimitsDispatchConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager, state, messages, runners := newDispatcherFixture(t, 2)
		oldFactory, oldRun := newAppTaskServiceClient, runAppTask
		t.Cleanup(func() { newAppTaskServiceClient, runAppTask = oldFactory, oldRun })
		release := make(chan struct{})
		active, maximum, completed := 0, 0, 0
		var mutex sync.Mutex
		newAppTaskServiceClient = func(context.Context, meta.App, string, taskspec.NATSMessage) appskeled.TaskServiceClientER {
			return nil
		}
		runAppTask = func(appskeled.TaskServiceClientER, appskeled.TaskRun, time.Duration) ex.Error {
			mutex.Lock()
			active++
			maximum = max(maximum, active)
			mutex.Unlock()
			<-release
			mutex.Lock()
			active--
			completed++
			mutex.Unlock()
			return nil
		}
		const total = 100
		for range total {
			messages.queue <- newDispatcherMessage()
		}
		go manager.dispatchTaskMessages(state)
		synctest.Wait()
		require.Equal(t, 2, active)
		require.Equal(t, int64(2), messages.reads.Load(), "full runners must stop Next calls, not just execution")
		require.Len(t, messages.queue, total-2)
		require.Len(t, runners[0].semaphore, 2)

		release <- struct{}{}
		synctest.Wait()
		require.Equal(t, 1, completed)
		require.Equal(t, int64(3), messages.reads.Load(), "completion must release capacity")
		close(release)
		synctest.Wait()
		require.Equal(t, total, completed)
		require.Equal(t, 2, maximum)
		// The single idle read reserves a slot, but must not count as app work.
		require.True(t, runners[0].instance.WaitDrain(t.Context()))
		manager.AfterAppStop()
		synctest.Wait()
		require.Empty(t, runners[0].semaphore)
	})
}

func TestDispatcherDrainCancelsIdleReadAndUsesRemainingRunner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager, state, messages, runners := newDispatcherFixture(t, 1, 1)
		go manager.dispatchTaskMessages(state)
		synctest.Wait()
		require.Equal(t, int64(1), messages.reads.Load())
		require.Len(t, runners[0].semaphore, 1)
		manager.OnDrain(runners[0].instance)
		synctest.Wait()
		require.Empty(t, runners[0].semaphore)
		require.Len(t, runners[1].semaphore, 1)
		require.Equal(t, int64(2), messages.reads.Load())
		manager.OnDrain(runners[1].instance)
		synctest.Wait()
		select {
		case <-state.done:
		default:
			t.Fatal("last runner drain left dispatcher alive")
		}
		require.Empty(t, runners[1].semaphore)
	})
}

func TestDispatcherCancellationWhileFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager, state, messages, runners := newDispatcherFixture(t, 1)
		ctx, cancel := context.WithCancel(manager.Context)
		manager.Context = ctx
		runners[0].semaphore <- struct{}{}
		go manager.dispatchTaskMessages(state)
		synctest.Wait()
		require.Zero(t, messages.reads.Load())
		cancel()
		synctest.Wait()
		select {
		case <-state.done:
		default:
			t.Fatal("capacity wait ignored cancellation")
		}
		<-runners[0].semaphore
	})
}

func TestManagerCompetesGloballyForTaskMessages(t *testing.T) {
	firstManager, firstCleanup := newTestManager(t)
	defer firstCleanup()
	secondManager, secondCleanup := newTestManager(t)
	defer secondCleanup()

	oldFactory := newAppTaskServiceClient
	oldRun := runAppTask
	defer func() {
		newAppTaskServiceClient = oldFactory
		runAppTask = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	secondHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppTaskServiceClient = func(_ context.Context, _ meta.App, endpoint string, _ taskspec.NATSMessage) appskeled.TaskServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppTaskClient{
			runTask: func(run appskeled.TaskRun) error {
				targetHooks.mutex.Lock()
				targetHooks.runs = append(targetHooks.runs, run)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				targetHooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppTask = func(client appskeled.TaskServiceClientER, run appskeled.TaskRun, timeout time.Duration) ex.Error {
		return client.RunTask(run)
	}

	firstAppInfo, err := meta.NewApp("demo.first", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.second", "1.0.0", "22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)

	firstEndpoint := testLocalAppEndpoint(8080)
	firstManager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:      firstAppInfo,
		TaskEndpoint: firstEndpoint + testPathTask,
		TaskRunners: []skeled.TaskRunnerRegistration{{
			TaskSkelName: "demo.user.SyncUserTask",
			TimeoutMs:    1000,
			Concurrency:  1,
		}},
	})
	secondEndpoint := testLocalAppEndpoint(8081)
	secondManager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:      secondAppInfo,
		TaskEndpoint: secondEndpoint + testPathTask,
		TaskRunners: []skeled.TaskRunnerRegistration{{
			TaskSkelName: "demo.user.SyncUserTask",
			TimeoutMs:    1000,
			Concurrency:  1,
		}},
	})

	firstManager.LaunchTask(skeled.TaskLaunch{
		Metadata: skeled.TaskLaunchMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		TaskSkelName:    "demo.user.SyncUserTask",
		TriggerSkelName: "demo.user.SyncUserTaskManualTrigger",
		ArgumentsJson:   `{"userId":"u1"}`,
	})

	select {
	case <-firstHooks.completed:
	case <-secondHooks.completed:
	case <-time.After(2 * time.Second):
		t.Fatal("competing task dispatch timeout")
	}

	firstHooks.mutex.Lock()
	firstCount := firstHooks.callCount
	firstHooks.mutex.Unlock()
	secondHooks.mutex.Lock()
	secondCount := secondHooks.callCount
	secondHooks.mutex.Unlock()
	assert.Equal(t, 1, firstCount+secondCount)
}

func TestManagerDispatchesTaskToSingleRunner(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppTaskServiceClient
	oldRun := runAppTask
	defer func() {
		newAppTaskServiceClient = oldFactory
		runAppTask = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	secondHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppTaskServiceClient = func(_ context.Context, _ meta.App, endpoint string, _ taskspec.NATSMessage) appskeled.TaskServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppTaskClient{
			runTask: func(run appskeled.TaskRun) error {
				targetHooks.mutex.Lock()
				targetHooks.runs = append(targetHooks.runs, run)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				targetHooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppTask = func(client appskeled.TaskServiceClientER, run appskeled.TaskRun, timeout time.Duration) ex.Error {
		return client.RunTask(run)
	}

	firstAppInfo, err := meta.NewApp("demo.first", "1.0.0", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.second", "1.0.0", "22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)

	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:      firstAppInfo,
		TaskEndpoint: testLocalAppEndpoint(8080) + testPathTask,
		TaskRunners: []skeled.TaskRunnerRegistration{{
			TaskSkelName: "demo.user.SyncUserTask",
			TimeoutMs:    1000,
			Concurrency:  1,
		}},
	})
	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:      secondAppInfo,
		TaskEndpoint: testLocalAppEndpoint(8081) + testPathTask,
		TaskRunners: []skeled.TaskRunnerRegistration{{
			TaskSkelName: "demo.user.SyncUserTask",
			TimeoutMs:    1000,
			Concurrency:  1,
		}},
	})

	manager.LaunchTask(skeled.TaskLaunch{
		Metadata: skeled.TaskLaunchMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		},
		TaskSkelName:    "demo.user.SyncUserTask",
		TriggerSkelName: "demo.user.SyncUserTaskManualTrigger",
		ArgumentsJson:   `{"userId":"u1"}`,
	})

	select {
	case <-firstHooks.completed:
	case <-secondHooks.completed:
	case <-time.After(2 * time.Second):
		t.Fatal("single-runner task dispatch timeout")
	}

	firstHooks.mutex.Lock()
	firstCount := firstHooks.callCount
	firstHooks.mutex.Unlock()
	secondHooks.mutex.Lock()
	secondCount := secondHooks.callCount
	secondHooks.mutex.Unlock()
	assert.Equal(t, 1, firstCount+secondCount)
}

func requireDispatchSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func testLocalAppEndpoint(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// These fakes exercise admission through the actual dispatcher, before Next.
// Unused jetstream.Msg methods deliberately remain unimplemented.
type dispatcherMessage struct {
	jetstream.Msg
	acked   atomic.Int64
	nacked  atomic.Int64
	renewed atomic.Int64
}

func newDispatcherMessage() *dispatcherMessage { return new(dispatcherMessage) }
func (m *dispatcherMessage) Data() []byte {
	return []byte(vcode.MustMarshalJsonS(taskspec.NATSMessage{TaskSkelName: "demo.task"}))
}
func (m *dispatcherMessage) Ack() error        { m.acked.Add(1); return nil }
func (m *dispatcherMessage) Nak() error        { m.nacked.Add(1); return nil }
func (m *dispatcherMessage) InProgress() error { m.renewed.Add(1); return nil }

type dispatcherMessages struct {
	queue   chan jetstream.Msg
	stopped chan struct{}
	stop    sync.Once
	reads   atomic.Int64
}

func (m *dispatcherMessages) Next(ctx context.Context) (jetstream.Msg, error) {
	m.reads.Add(1)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.stopped:
		return nil, jetstream.ErrMsgIteratorClosed
	case msg := <-m.queue:
		return msg, nil
	}
}
func (m *dispatcherMessages) Stop() { m.stop.Do(func() { close(m.stopped) }) }
func newDispatcherFixture(t *testing.T, capacities ...int) (*Manager, *_TaskSubscription, *dispatcherMessages, []*_TaskRunnerState) {
	t.Helper()
	messages := new(dispatcherMessages{queue: make(chan jetstream.Msg, 100), stopped: make(chan struct{})})
	state := new(_TaskSubscription{messages: messages, runnerByApp: map[string]*_TaskRunnerState{}, changed: make(chan struct{}), done: make(chan struct{})})
	manager := new(Manager{
		Context:                    t.Context(),
		runnerByAppInstanceID:      map[string]map[string]*_TaskRunnerState{},
		subscriptionByTaskSkelName: map[string]*_TaskSubscription{"demo.task": state},
	})
	runners := make([]*_TaskRunnerState, 0, len(capacities))
	for i, capacity := range capacities {
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		ctx, cancel := context.WithCancel(manager.Context)
		t.Cleanup(cancel)
		runner := new(_TaskRunnerState{
			instance:      new(minder.AppInstance{AppInfo: meta.MustNewApp("demo.app", "1.0.0", id)}),
			appInstanceID: id, registration: skeled.TaskRunnerRegistration{TaskSkelName: "demo.task", Concurrency: capacity},
			semaphore: make(chan struct{}, capacity), readContext: ctx, cancelRead: cancel, subscription: state,
		})
		runners = append(runners, runner)
		state.runnerByApp[id] = runner
		manager.runnerByAppInstanceID[id] = map[string]*_TaskRunnerState{"demo.task": runner}
	}
	return manager, state, messages, runners
}

func TestTaskMessageRenewalStopsBeforeFinalAck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		msg := newDispatcherMessage()
		stop := keepTaskMessageAlive(msg)
		synctest.Wait()
		time.Sleep(35 * time.Second)
		synctest.Wait()
		require.Equal(t, int64(3), msg.renewed.Load())
		stop()
		require.NoError(t, msg.Ack())
		stop()
		time.Sleep(20 * time.Second)
		synctest.Wait()
		require.Equal(t, int64(3), msg.renewed.Load())
	})
}

func TestDispatcherAcknowledgementReleasesCapacity(t *testing.T) {
	for _, test := range []struct {
		name    string
		noRetry bool
		fail    bool
	}{
		{name: "success"},
		{name: "retry", fail: true},
		{name: "no retry", noRetry: true, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				manager, state, messages, runners := newDispatcherFixture(t, 1)
				runners[0].registration.NoRetry = test.noRetry
				oldFactory, oldRun := newAppTaskServiceClient, runAppTask
				t.Cleanup(func() { newAppTaskServiceClient, runAppTask = oldFactory, oldRun })
				newAppTaskServiceClient = func(context.Context, meta.App, string, taskspec.NATSMessage) appskeled.TaskServiceClientER {
					return nil
				}
				runAppTask = func(appskeled.TaskServiceClientER, appskeled.TaskRun, time.Duration) ex.Error {
					if test.fail {
						return ex.New(ex.Internal, "dispatch test failure")
					}
					return nil
				}
				first, second := newDispatcherMessage(), newDispatcherMessage()
				messages.queue <- first
				messages.queue <- second
				go manager.dispatchTaskMessages(state)
				synctest.Wait()
				for _, msg := range []*dispatcherMessage{first, second} {
					if test.fail && !test.noRetry {
						require.Equal(t, int64(1), msg.nacked.Load())
						require.Zero(t, msg.acked.Load())
					} else {
						require.Equal(t, int64(1), msg.acked.Load())
						require.Zero(t, msg.nacked.Load())
					}
				}
				manager.AfterAppStop()
				synctest.Wait()
				require.Empty(t, runners[0].semaphore)
			})
		})
	}
}

func TestDispatcherSkipsSaturatedRunner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager, state, _, runners := newDispatcherFixture(t, 1, 2)
		runners[0].semaphore <- struct{}{}
		first := manager.nextAvailableRunner(state)
		second := manager.nextAvailableRunner(state)
		require.Same(t, runners[1], first)
		require.Same(t, runners[1], second)
		manager.releaseTaskSlot(first)
		manager.releaseTaskSlot(second)
		<-runners[0].semaphore
		require.Empty(t, runners[1].semaphore)
	})
}
