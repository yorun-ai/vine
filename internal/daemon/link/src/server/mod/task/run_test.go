package task

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
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

	hooks := &_ManagerDispatchHooks{}
	newAppTaskServiceClient = func(context.Context, runtime.App, string, taskspec.NATSMessage) appskeled.TaskServiceClientER {
		return &_ManagerAppTaskClient{
			runTask: func(run appskeled.TaskRun) error {
				hooks.mutex.Lock()
				hooks.runs = append(hooks.runs, run)
				hooks.mutex.Unlock()
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

	require.Eventually(t, func() bool {
		hooks.mutex.Lock()
		defer hooks.mutex.Unlock()
		return len(hooks.runs) == 1
	}, 2*time.Second, 10*time.Millisecond)

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
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppTaskServiceClient
	oldRun := runAppTask
	defer func() {
		newAppTaskServiceClient = oldFactory
		runAppTask = oldRun
	}()

	hooks := &_ManagerDispatchHooks{
		startedChan: make(chan struct{}, 2),
		releaseChan: make(chan struct{}),
	}
	newAppTaskServiceClient = func(context.Context, runtime.App, string, taskspec.NATSMessage) appskeled.TaskServiceClientER {
		return &_ManagerAppTaskClient{
			runTask: func(run appskeled.TaskRun) error {
				hooks.mutex.Lock()
				hooks.runs = append(hooks.runs, run)
				hooks.mutex.Unlock()
				hooks.startedChan <- struct{}{}
				<-hooks.releaseChan
				return nil
			},
		}
	}
	runAppTask = func(client appskeled.TaskServiceClientER, run appskeled.TaskRun, timeout time.Duration) ex.Error {
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
			TimeoutMs:    1000,
			Concurrency:  1,
		}},
	})

	firstLaunch := skeled.TaskLaunch{
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
	}
	secondLaunch := firstLaunch
	secondLaunch.Metadata.TraceId = "trace-2"
	secondLaunch.ArgumentsJson = `{"userId":"u2"}`

	manager.LaunchTask(firstLaunch)
	manager.LaunchTask(secondLaunch)

	select {
	case <-hooks.startedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("first task dispatch timeout")
	}

	select {
	case <-hooks.startedChan:
		t.Fatal("second task started before first task finished")
	case <-time.After(100 * time.Millisecond):
	}

	hooks.releaseChan <- struct{}{}

	select {
	case <-hooks.startedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("second task dispatch timeout")
	}

	hooks.releaseChan <- struct{}{}

	require.Eventually(t, func() bool {
		hooks.mutex.Lock()
		defer hooks.mutex.Unlock()
		return len(hooks.runs) == 2
	}, 2*time.Second, 10*time.Millisecond)
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

	firstHooks := &_ManagerDispatchHooks{}
	secondHooks := &_ManagerDispatchHooks{}
	newAppTaskServiceClient = func(_ context.Context, _ runtime.App, endpoint string, _ taskspec.NATSMessage) appskeled.TaskServiceClientER {
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

	require.Eventually(t, func() bool {
		firstHooks.mutex.Lock()
		firstCount := firstHooks.callCount
		firstHooks.mutex.Unlock()
		secondHooks.mutex.Lock()
		secondCount := secondHooks.callCount
		secondHooks.mutex.Unlock()
		return firstCount+secondCount == 1
	}, 2*time.Second, 10*time.Millisecond)

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

	firstHooks := &_ManagerDispatchHooks{}
	secondHooks := &_ManagerDispatchHooks{}
	newAppTaskServiceClient = func(_ context.Context, _ runtime.App, endpoint string, _ taskspec.NATSMessage) appskeled.TaskServiceClientER {
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

	require.Eventually(t, func() bool {
		firstHooks.mutex.Lock()
		firstCount := firstHooks.callCount
		firstHooks.mutex.Unlock()
		secondHooks.mutex.Lock()
		secondCount := secondHooks.callCount
		secondHooks.mutex.Unlock()
		return firstCount+secondCount == 1
	}, 2*time.Second, 10*time.Millisecond)

	firstHooks.mutex.Lock()
	firstCount := firstHooks.callCount
	firstHooks.mutex.Unlock()
	secondHooks.mutex.Lock()
	secondCount := secondHooks.callCount
	secondHooks.mutex.Unlock()
	assert.Equal(t, 1, firstCount+secondCount)
}

func testLocalAppEndpoint(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
