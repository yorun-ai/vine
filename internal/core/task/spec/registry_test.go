package spec

import (
	"reflect"
	"sync"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
)

type testInfoTaskLauncher interface {
	mustBeTestInfoTaskLauncher()
}

type defaultTestInfoTaskLauncher struct{}

func (*defaultTestInfoTaskLauncher) mustBeTestInfoTaskLauncher() {}

type testInfoTaskRunner interface {
	RunForGroup(testInfoTaskArguments)
	mustBeTestInfoTaskRunner()
}

type defaultTestInfoTaskRunner struct{}

func (*defaultTestInfoTaskRunner) RunForGroup(testInfoTaskArguments) {}
func (*defaultTestInfoTaskRunner) mustBeTestInfoTaskRunner()         {}

type testInfoTaskRunnerER interface {
	RunForGroup(testInfoTaskArguments) ex.Error
	mustBeTestInfoTaskRunnerER()
}

type _WrapperTestInfoTaskRunnerER struct {
	defaultTestInfoTaskRunner
	runnerImpl testInfoTaskRunner
}

func newWrapperTestInfoTaskRunnerER(runnerImpl testInfoTaskRunner) testInfoTaskRunnerER {
	return &_WrapperTestInfoTaskRunnerER{runnerImpl: runnerImpl}
}

func (r *_WrapperTestInfoTaskRunnerER) runner() testInfoTaskRunner {
	if r.runnerImpl == nil {
		return &r.defaultTestInfoTaskRunner
	}
	return r.runnerImpl
}

func (r *_WrapperTestInfoTaskRunnerER) RunForGroup(arguments testInfoTaskArguments) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().RunForGroup(arguments)
	return
}

func (*_WrapperTestInfoTaskRunnerER) mustBeTestInfoTaskRunnerER() {}

type defaultTestInfoTaskRunnerER struct {
	_WrapperTestInfoTaskRunnerER
}

type testInfoTaskArguments struct {
	GroupId int
}

var testInfoTaskRegisterOnce sync.Once

func ensureInfoTaskRegistered() {
	testInfoTaskRegisterOnce.Do(func() {
		Register(newTestInfoTaskSpec())
	})
}

func newTestInfoTaskSpec() *TaskSpec {
	return &TaskSpec{
		Name:                "TestInfoTask",
		SkelName:            "test.info.task",
		RunnerType:          reflect.TypeFor[testInfoTaskRunner](),
		DefaultRunnerType:   reflect.TypeFor[*defaultTestInfoTaskRunner](),
		ERRunnerType:        reflect.TypeFor[testInfoTaskRunnerER](),
		WrapperERRunnerCtor: newWrapperTestInfoTaskRunnerER,
		DefaultERRunnerType: reflect.TypeFor[*defaultTestInfoTaskRunnerER](),
		LauncherType:        reflect.TypeFor[testInfoTaskLauncher](),
		LauncherCtor:        func(*struct{}) testInfoTaskLauncher { return &defaultTestInfoTaskLauncher{} },
		Triggers: []*TriggerSpec{{
			Name:               "ForGroup",
			SkelName:           "forGroup",
			LauncherMethodName: "LaunchForGroup",
			RunnerMethodName:   "RunForGroup",
			ArgumentsType:      reflect.TypeFor[testInfoTaskArguments](),
			ArgumentsSensitive: true,
		}},
	}
}

func TestGetTriggerInfoReturnsRegisteredTrigger(t *testing.T) {
	registry := NewRegistry()
	registry.Register(newTestInfoTaskSpec())

	triggerInfo, ok := registry.GetTriggerInfo("test.info.task", "forGroup")
	if !ok {
		t.Fatal("expected trigger info")
	}
	if triggerInfo.Name() != "ForGroup" {
		t.Fatalf("unexpected trigger info: %+v", triggerInfo)
	}
	if triggerInfo.LauncherMethodName() != "LaunchForGroup" {
		t.Fatalf("unexpected launcher method name: %+v", triggerInfo)
	}
	if triggerInfo.RunnerMethodName() != "RunForGroup" {
		t.Fatalf("unexpected runner method name: %+v", triggerInfo)
	}
	if !triggerInfo.ArgumentsSensitive() {
		t.Fatal("expected trigger arguments to be sensitive")
	}
}
