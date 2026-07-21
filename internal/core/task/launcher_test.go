package task

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.yorun.ai/vine/internal/core/ex"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/core/task/spec"
)

func testLauncherContext() meta.Context {
	return &meta.ContextImpl{
		Context:        context.Background(),
		TraceValue:     meta.InitialTrace(),
		InitiatorValue: nil,
		ActorValue:     nil,
	}
}

func testLauncherLogger() *logger.Logger {
	return logger.NewLogger(logger.GlobalOption())
}

type testLauncherTaskClient struct {
	launch linkskeled.TaskLaunch
}

func (c *testLauncherTaskClient) LaunchTask(launch linkskeled.TaskLaunch, _ivOpts ...rpcclient.InvokeOption) {
	c.launch = launch
}

type testLauncherTaskArguments struct {
	StartAt time.Time
}

type testLauncherTaskRunner interface {
	RunAtTime(testLauncherTaskArguments)
	mustBeTestLauncherTaskRunner()
}

type defaultTestLauncherTaskRunner struct{}

func (*defaultTestLauncherTaskRunner) RunAtTime(testLauncherTaskArguments) {}
func (*defaultTestLauncherTaskRunner) mustBeTestLauncherTaskRunner()       {}

type testLauncherTaskRunnerER interface {
	RunAtTime(testLauncherTaskArguments) ex.Error
	mustBeTestLauncherTaskRunnerER()
}

type _WrapperTestLauncherTaskRunnerER struct {
	defaultTestLauncherTaskRunner
	runnerImpl testLauncherTaskRunner
}

func newWrapperTestLauncherTaskRunnerER(runnerImpl testLauncherTaskRunner) testLauncherTaskRunnerER {
	return &_WrapperTestLauncherTaskRunnerER{runnerImpl: runnerImpl}
}

func (r *_WrapperTestLauncherTaskRunnerER) runner() testLauncherTaskRunner {
	if r.runnerImpl == nil {
		return &r.defaultTestLauncherTaskRunner
	}
	return r.runnerImpl
}

func (r *_WrapperTestLauncherTaskRunnerER) RunAtTime(arguments testLauncherTaskArguments) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().RunAtTime(arguments)
	return
}

func (*_WrapperTestLauncherTaskRunnerER) mustBeTestLauncherTaskRunnerER() {}

type defaultTestLauncherTaskRunnerER struct {
	_WrapperTestLauncherTaskRunnerER
}

func testLauncherTriggerInfo() spec.TriggerInfo {
	return spec.ConvertSpecToInfoForTest(&spec.TaskSpec{
		Name:                "LauncherTask",
		SkelName:            "launcher.task",
		RunnerType:          reflect.TypeOf((*testLauncherTaskRunner)(nil)).Elem(),
		DefaultRunnerType:   reflect.TypeOf(&defaultTestLauncherTaskRunner{}),
		ERRunnerType:        reflect.TypeOf((*testLauncherTaskRunnerER)(nil)).Elem(),
		WrapperERRunnerCtor: newWrapperTestLauncherTaskRunnerER,
		DefaultERRunnerType: reflect.TypeOf(&defaultTestLauncherTaskRunnerER{}),
		Triggers: []*spec.TriggerSpec{{
			Name:               "AtTime",
			SkelName:           "atTime",
			LauncherMethodName: "LaunchAtTime",
			RunnerMethodName:   "RunAtTime",
			ArgumentsType:      reflect.TypeOf(testLauncherTaskArguments{}),
		}},
	}).Triggers()[0]
}

func TestTaskLauncherBuildRequestInjectsMetaFields(t *testing.T) {
	taskClient := &testLauncherTaskClient{}
	launcher := NewLauncher(LauncherOption{
		Context:    testLauncherContext(),
		Logger:     testLauncherLogger(),
		ClientApp:  meta.MustNewApp("test.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000"),
		TaskClient: taskClient,
	})

	launch := launcher.buildLaunch(testLauncherTriggerInfo(), &testLauncherTaskArguments{}, nil)
	if launch.Metadata.TraceId != launcher.context.Trace().Id() {
		t.Fatalf("unexpected trace id: got %s want %s", launch.Metadata.TraceId, launcher.context.Trace().Id())
	}
	if launch.Metadata.AppName != launcher.clientApp.Name() {
		t.Fatalf("unexpected client app: got %s want %s", launch.Metadata.AppName, launcher.clientApp.Name())
	}
	if launch.Metadata.AppInstanceId != skel.NewUUID(uuid.MustParse(launcher.clientApp.InstanceId())) {
		t.Fatalf("unexpected client app instance id: got %s want %s", launch.Metadata.AppInstanceId, launcher.clientApp.InstanceId())
	}
}

func TestTaskLauncherLaunchUsesTaskClient(t *testing.T) {
	taskClient := &testLauncherTaskClient{}
	launcher := NewLauncher(LauncherOption{
		Context:    testLauncherContext(),
		Logger:     testLauncherLogger(),
		ClientApp:  meta.MustNewApp("test.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000"),
		TaskClient: taskClient,
	})

	launcher.Launch(testLauncherTriggerInfo(), &testLauncherTaskArguments{StartAt: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)})
	if taskClient.launch.TaskSkelName != "launcher.task" {
		t.Fatalf("unexpected task skel name: %s", taskClient.launch.TaskSkelName)
	}
	if taskClient.launch.TriggerSkelName != "atTime" {
		t.Fatalf("unexpected trigger skel name: %s", taskClient.launch.TriggerSkelName)
	}
	if taskClient.launch.ArgumentsJson == "" {
		t.Fatal("expected arguments json")
	}
}

func TestNewTaskLauncherPanicsWhenContextIsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = NewLauncher(LauncherOption{
		Logger:     testLauncherLogger(),
		TaskClient: &testLauncherTaskClient{},
	})
}

func TestNewTaskLauncherPanicsWhenLoggerIsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = NewLauncher(LauncherOption{
		Context:    testLauncherContext(),
		TaskClient: &testLauncherTaskClient{},
	})
}

func TestNewTaskLauncherPanicsWhenTaskClientIsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = NewLauncher(LauncherOption{
		Context: testLauncherContext(),
		Logger:  testLauncherLogger(),
	})
}

func TestNewTaskLauncherStoresConfiguredValues(t *testing.T) {
	taskClient := &testLauncherTaskClient{}
	launcher := NewLauncher(LauncherOption{
		Context:    testLauncherContext(),
		Logger:     testLauncherLogger(),
		TaskClient: taskClient,
	})

	if launcher.context == nil || launcher.logger == nil {
		t.Fatalf("unexpected launcher: %+v", launcher)
	}
	if launcher.taskClient != taskClient {
		t.Fatal("expected task client to be stored")
	}
}
