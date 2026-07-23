package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/core/task/spec"
)

type testRunnerArguments struct {
	GroupId int
}

type testRunnerTaskRunner interface {
	RunForGroup(testRunnerArguments)
	mustBeTestRunnerTaskRunner()
}

type defaultTestRunnerTaskRunner struct{}

func (*defaultTestRunnerTaskRunner) RunForGroup(testRunnerArguments) {}
func (*defaultTestRunnerTaskRunner) mustBeTestRunnerTaskRunner()     {}

type testRunnerTaskRunnerER interface {
	RunForGroup(testRunnerArguments) ex.Error
	mustBeTestRunnerTaskRunnerER()
}

type _WrapperTestRunnerTaskRunnerER struct {
	defaultTestRunnerTaskRunner
	runnerImpl testRunnerTaskRunner
}

func newWrapperTestRunnerTaskRunnerER(runnerImpl testRunnerTaskRunner) testRunnerTaskRunnerER {
	return &_WrapperTestRunnerTaskRunnerER{runnerImpl: runnerImpl}
}

func (r *_WrapperTestRunnerTaskRunnerER) runner() testRunnerTaskRunner {
	if r.runnerImpl == nil {
		return &r.defaultTestRunnerTaskRunner
	}
	return r.runnerImpl
}

func (r *_WrapperTestRunnerTaskRunnerER) RunForGroup(arguments testRunnerArguments) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runner().RunForGroup(arguments)
	return
}

func (*_WrapperTestRunnerTaskRunnerER) mustBeTestRunnerTaskRunnerER() {}

type defaultTestRunnerTaskRunnerER struct {
	_WrapperTestRunnerTaskRunnerER
}

type testRunnerImpl struct {
	defaultTestRunnerTaskRunner
}

func (*testRunnerImpl) RunForGroup(testRunnerArguments) {}

type _RunnerRecorderExecutor struct {
	taskContext spec.Context
	triggerImpl spec.TriggerImpl
	args        []any
	err         ex.Error
	panicV      any
}

func (*_RunnerRecorderExecutor) Init(spec.ImplDict) {}

func (e *_RunnerRecorderExecutor) Execute(taskContext spec.Context, triggerImpl spec.TriggerImpl, args []any) ex.Error {
	if e.panicV != nil {
		panic(e.panicV)
	}
	e.taskContext = taskContext
	e.triggerImpl = triggerImpl
	e.args = args
	return e.err
}

var testRunnerRegisterOnce sync.Once

func ensureRunnerTaskRegistered() {
	testRunnerRegisterOnce.Do(func() {
		spec.Register(&spec.TaskSpec{
			Name:                "RunnerTask",
			SkelName:            "runner.task",
			RunnerType:          reflect.TypeOf((*testRunnerTaskRunner)(nil)).Elem(),
			DefaultRunnerType:   reflect.TypeOf(&defaultTestRunnerTaskRunner{}),
			ERRunnerType:        reflect.TypeOf((*testRunnerTaskRunnerER)(nil)).Elem(),
			WrapperERRunnerCtor: newWrapperTestRunnerTaskRunnerER,
			DefaultERRunnerType: reflect.TypeOf(&defaultTestRunnerTaskRunnerER{}),
			Triggers: []*spec.TriggerSpec{{
				Name:               "ForGroup",
				SkelName:           "forGroup",
				LauncherMethodName: "LaunchForGroup",
				RunnerMethodName:   "RunForGroup",
				ArgumentsType:      reflect.TypeOf(testRunnerArguments{}),
			}},
		})
	})
}

func testRunnerTriggerInfo() spec.TriggerInfo {
	triggerInfo, ok := spec.GetTriggerInfo("runner.task", "forGroup")
	if !ok {
		panic("runner task trigger not registered")
	}
	return triggerInfo
}

func TestServerResolvesTriggerImplByInfo(t *testing.T) {
	ensureRunnerTaskRegistered()

	executor := &_RunnerRecorderExecutor{}
	server := NewServer(Option{
		ImplTypes: []reflect.Type{reflect.TypeOf(&testRunnerImpl{})},
		Executor:  executor,
	})
	baseTrace := meta.InitialTrace()
	triggerImpl, err := server.implDict.GetTriggerImpl("runner.task", "forGroup")
	if err != nil {
		t.Fatalf("GetTriggerImpl() error = %v", err)
	}

	err = server.runTask(&spec.RunImpl{
		ContextValue: &spec.ContextImpl{
			ContextImpl: meta.ContextImpl{
				Context:    context.Background(),
				TraceValue: baseTrace,
			},
		},
		TriggerImplValue: triggerImpl,
		TriggerInfoValue: testRunnerTriggerInfo(),
		ArgumentsValue:   &testRunnerArguments{GroupId: 9},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executor.triggerImpl == nil {
		t.Fatal("expected trigger impl")
	}
	if executor.taskContext.Trace().ParentSpan() != "" {
		t.Fatalf("unexpected parent span: got=%s want empty", executor.taskContext.Trace().ParentSpan())
	}
}

func TestServerReturnsInvalidTaskWhenTriggerNotRegistered(t *testing.T) {
	ensureRunnerTaskRegistered()

	logPath := filepath.Join(t.TempDir(), "task-rejected.jsonl")
	server := NewServer(Option{
		ImplTypes: []reflect.Type{reflect.TypeOf(&testRunnerImpl{})},
		Executor:  &_RunnerRecorderExecutor{},
		Logger: logger.NewLogger(&logger.Option{
			Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: logPath,
		}),
	})

	err := server.RunTask(context.Background(), appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       meta.InitialTrace().Id(),
			TraceSpan:     meta.InitialTrace().Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		TaskSkelName:    "missing.task",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{}`,
	})
	if err == nil || err.Code() != ex.InvalidTask {
		t.Fatalf("unexpected error: %#v", err)
	}
	logBytes, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read rejection log: %v", readErr)
	}
	lines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")
	if len(lines) != 1 {
		t.Fatalf("execution-before rejection should emit one record: %s", logBytes)
	}
	var record map[string]any
	if decodeErr := json.Unmarshal([]byte(lines[0]), &record); decodeErr != nil {
		t.Fatalf("decode rejection log: %v", decodeErr)
	}
	if record["msg"] != "task runner handle rejected" || record["level"] != "DEBUG" || record["code"] != string(ex.InvalidTask) {
		t.Fatalf("unexpected rejection record: %#v", record)
	}
	if record["taskSkel"] != "missing.task" || record["taskTriggerSkel"] != "forGroup" {
		t.Fatalf("rejection record must preserve main Task field names: %#v", record)
	}
}

func TestServerConvertsRecoveredPanicToInternalError(t *testing.T) {
	ensureRunnerTaskRegistered()

	server := NewServer(Option{
		ImplTypes: []reflect.Type{reflect.TypeOf(&testRunnerImpl{})},
		Executor: &_RunnerRecorderExecutor{
			panicV: "boom",
		},
	})
	triggerImpl, getErr := server.implDict.GetTriggerImpl("runner.task", "forGroup")
	if getErr != nil {
		t.Fatalf("GetTriggerImpl() error = %v", getErr)
	}

	err := server.runTask(&spec.RunImpl{
		ContextValue: &spec.ContextImpl{
			ContextImpl: meta.ContextImpl{
				Context:    context.Background(),
				TraceValue: meta.InitialTrace(),
			},
		},
		TriggerImplValue: triggerImpl,
		TriggerInfoValue: testRunnerTriggerInfo(),
		ArgumentsValue:   &testRunnerArguments{},
	})
	if err == nil || err.Code() != ex.Internal {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestServerRunTaskResolvesAndRunsTrigger(t *testing.T) {
	ensureRunnerTaskRegistered()

	executor := &_RunnerRecorderExecutor{}
	server := NewServer(Option{
		ImplTypes: []reflect.Type{reflect.TypeOf(&testRunnerImpl{})},
		Executor:  executor,
	})
	baseTrace := meta.InitialTrace()

	err := server.RunTask(context.Background(), appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       baseTrace.Id(),
			TraceSpan:     baseTrace.Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		TaskSkelName:    "runner.task",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{"groupId":11}`,
	})
	if err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}
	if executor.triggerImpl == nil {
		t.Fatal("expected trigger impl")
	}
	if executor.taskContext.Trace().Id() != baseTrace.Id() {
		t.Fatalf("unexpected trace id: got=%s want=%s", executor.taskContext.Trace().Id(), baseTrace.Id())
	}
	if executor.taskContext.Trace().ParentSpan() != baseTrace.Span() {
		t.Fatalf("unexpected parent span: got=%s want=%s", executor.taskContext.Trace().ParentSpan(), baseTrace.Span())
	}
	if got := executor.args[0].(int); got != 11 {
		t.Fatalf("unexpected group id: got=%d want=%d", got, 11)
	}
}
