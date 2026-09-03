package task

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/task/spec"
)

type _ExecutorRecorder struct {
	TaskCtx     spec.Context
	TriggerInfo spec.TriggerInfo
}

type testContainerExecutorArguments struct {
	GroupId int
}

type testContainerExecutorRunner interface {
	RunForGroup(testContainerExecutorArguments)
	mustBeTestContainerExecutorRunner()
}

type defaultTestContainerExecutorRunner struct{}

func (*defaultTestContainerExecutorRunner) RunForGroup(testContainerExecutorArguments) {}
func (*defaultTestContainerExecutorRunner) mustBeTestContainerExecutorRunner()         {}

type testContainerExecutorRunnerER interface {
	RunForGroup(testContainerExecutorArguments) ex.Error
	mustBeTestContainerExecutorRunnerER()
}

type _WrapperTestContainerExecutorRunnerER struct {
	defaultTestContainerExecutorRunner
	runnerImpl testContainerExecutorRunner
}

func newWrapperTestContainerExecutorRunnerER(runnerImpl testContainerExecutorRunner) testContainerExecutorRunnerER {
	return &_WrapperTestContainerExecutorRunnerER{runnerImpl: runnerImpl}
}

func (r *_WrapperTestContainerExecutorRunnerER) runner() testContainerExecutorRunner {
	if r.runnerImpl == nil {
		return &r.defaultTestContainerExecutorRunner
	}
	return r.runnerImpl
}

func (r *_WrapperTestContainerExecutorRunnerER) RunForGroup(arguments testContainerExecutorArguments) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().RunForGroup(arguments)
	return
}

func (*_WrapperTestContainerExecutorRunnerER) mustBeTestContainerExecutorRunnerER() {}

type defaultTestContainerExecutorRunnerER struct {
	_WrapperTestContainerExecutorRunnerER
}

type testContainerExecutorImpl struct {
	defaultTestContainerExecutorRunner
	Recorder *_ExecutorRecorder `inject:""`
}

func (r *testContainerExecutorImpl) RunForGroup(testContainerExecutorArguments) {
	r.Recorder.TriggerInfo = nil
}

type testContainerExecutorImplWithSeed struct {
	defaultTestContainerExecutorRunner
	Recorder *_ExecutorRecorder `inject:""`
	TaskCtx  spec.Context       `inject:""`
	Trigger  spec.TriggerInfo   `inject:""`
}

func (r *testContainerExecutorImplWithSeed) RunForGroup(testContainerExecutorArguments) {
	r.Recorder.TaskCtx = r.TaskCtx
	r.Recorder.TriggerInfo = r.Trigger
}

type testContainerExecutorERImpl struct {
	defaultTestContainerExecutorRunnerER
	Recorder *_ExecutorRecorder `inject:""`
	Trigger  spec.TriggerInfo   `inject:""`
}

func (r *testContainerExecutorERImpl) RunForGroup(testContainerExecutorArguments) ex.Error {
	r.Recorder.TriggerInfo = r.Trigger
	return nil
}

var registerContainerExecutorTaskOnce sync.Once

func ensureContainerExecutorTaskRegistered() {
	registerContainerExecutorTaskOnce.Do(func() {
		spec.Register(&spec.TaskSpec{
			Name:                "ContainerExecutorTask",
			SkelName:            "container.executor.task",
			RunnerType:          reflect.TypeFor[testContainerExecutorRunner](),
			DefaultRunnerType:   reflect.TypeFor[*defaultTestContainerExecutorRunner](),
			ERRunnerType:        reflect.TypeFor[testContainerExecutorRunnerER](),
			WrapperERRunnerCtor: newWrapperTestContainerExecutorRunnerER,
			DefaultERRunnerType: reflect.TypeFor[*defaultTestContainerExecutorRunnerER](),
			Triggers: []*spec.TriggerSpec{{
				Name:               "ForGroup",
				SkelName:           "forGroup",
				LauncherMethodName: "LaunchForGroup",
				RunnerMethodName:   "RunForGroup",
				ArgumentsType:      reflect.TypeFor[testContainerExecutorArguments](),
			}},
		})
	})
}

func testContainerExecutorTriggerImpl(t *testing.T, implType reflect.Type) spec.TriggerImpl {
	t.Helper()
	ensureContainerExecutorTaskRegistered()
	implDict := spec.NewImplDict()
	implDict.Add(implType)
	triggerImpl, err := implDict.GetTriggerImpl("container.executor.task", "forGroup")
	if err != nil {
		t.Fatalf("GetTriggerImpl() error = %v", err)
	}
	return triggerImpl
}

func TestExecutorSeedsContextAndTriggerInfo(t *testing.T) {
	ensureContainerExecutorTaskRegistered()
	recorder := &_ExecutorRecorder{}
	executor := NewContainerExecutor(nil, []di.BindApplier{
		func(b *di.Binder) {
			b.Bind(di.T[*_ExecutorRecorder]()).ToInstance(recorder)
		},
	})
	implDict := spec.NewImplDict()
	implDict.Add(reflect.TypeFor[*testContainerExecutorImplWithSeed]())
	executor.Init(*implDict)

	taskCtx := &spec.ContextImpl{
		Context:    context.Background(),
		TraceValue: meta.InitialTrace(),
	}
	triggerImpl := testContainerExecutorTriggerImpl(t, reflect.TypeFor[*testContainerExecutorImplWithSeed]())
	err := executor.Execute(taskCtx, triggerImpl, []any{testContainerExecutorArguments{GroupId: 7}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if recorder.TaskCtx != taskCtx {
		t.Fatalf("unexpected task context: %#v", recorder.TaskCtx)
	}
	if recorder.TriggerInfo != triggerImpl.Info() {
		t.Fatalf("unexpected trigger info: %#v", recorder.TriggerInfo)
	}
}

func TestExecutorRunsERImplementation(t *testing.T) {
	ensureContainerExecutorTaskRegistered()
	recorder := &_ExecutorRecorder{}
	executor := NewContainerExecutor(nil, []di.BindApplier{
		func(b *di.Binder) {
			b.Bind(di.T[*_ExecutorRecorder]()).ToInstance(recorder)
		},
	})
	implDict := spec.NewImplDict()
	implDict.Add(reflect.TypeFor[*testContainerExecutorERImpl]())
	executor.Init(*implDict)

	triggerImpl := testContainerExecutorTriggerImpl(t, reflect.TypeFor[*testContainerExecutorERImpl]())
	err := executor.Execute(&spec.ContextImpl{}, triggerImpl, []any{testContainerExecutorArguments{GroupId: 7}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if recorder.TriggerInfo != triggerImpl.Info() {
		t.Fatalf("unexpected trigger info: %#v", recorder.TriggerInfo)
	}
}
