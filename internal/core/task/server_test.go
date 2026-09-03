package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"uuid"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/core/task/spec"
)

type testRunnerArguments struct {
	GroupId int
}

func testTaskServerApp() meta.App {
	return meta.MustNewApp("test.app", "1.0.0", "123e4567-e89b-12d3-a456-426614174011")
}

func TestNewServerRequiresApp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing App to panic")
		}
	}()
	NewServer(Option{})
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

type benchmarkTaskRunner interface {
	RunForGroup(groupId int)
	mustBeBenchmarkTaskRunner()
}

type defaultBenchmarkTaskRunner struct{}

func (*defaultBenchmarkTaskRunner) RunForGroup(int)            {}
func (*defaultBenchmarkTaskRunner) mustBeBenchmarkTaskRunner() {}

type benchmarkTaskRunnerER interface {
	RunForGroup(groupId int) ex.Error
	mustBeBenchmarkTaskRunnerER()
}

type wrapperBenchmarkTaskRunnerER struct {
	defaultBenchmarkTaskRunner
	runnerImpl benchmarkTaskRunner
}

func newWrapperBenchmarkTaskRunnerER(runnerImpl benchmarkTaskRunner) benchmarkTaskRunnerER {
	return &wrapperBenchmarkTaskRunnerER{runnerImpl: runnerImpl}
}

func (r *wrapperBenchmarkTaskRunnerER) RunForGroup(groupId int) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	r.runnerImpl.RunForGroup(groupId)
	return
}

func (*wrapperBenchmarkTaskRunnerER) mustBeBenchmarkTaskRunnerER() {}

type defaultBenchmarkTaskRunnerER struct {
	wrapperBenchmarkTaskRunnerER
}

type benchmarkTaskRunnerImpl struct {
	defaultBenchmarkTaskRunner
}

func (*benchmarkTaskRunnerImpl) RunForGroup(int) {}

type _RunnerRecorderExecutor struct {
	taskContext   spec.Context
	triggerImpl   spec.TriggerImpl
	args          []any
	err           ex.Error
	panicV        any
	profileLabels map[string]string
}

func (*_RunnerRecorderExecutor) Init(spec.ImplDict) {}

func (e *_RunnerRecorderExecutor) Execute(taskContext spec.Context, triggerImpl spec.TriggerImpl, args []any) ex.Error {
	if e.panicV != nil {
		panic(e.panicV)
	}
	e.taskContext = taskContext
	e.triggerImpl = triggerImpl
	e.args = args
	if e.profileLabels != nil {
		for _, key := range []string{"vine.app", "vine.protocol", "vine.task", "vine.trigger"} {
			if value, ok := pprof.Label(taskContext, key); ok {
				e.profileLabels[key] = value
			}
		}
	}
	return e.err
}

var testRunnerRegisterOnce sync.Once

var benchmarkTaskRegisterOnce sync.Once

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

func ensureBenchmarkTaskRegistered() {
	benchmarkTaskRegisterOnce.Do(func() {
		spec.Register(&spec.TaskSpec{
			Name:                "BenchmarkTask",
			SkelName:            "benchmark.task",
			RunnerType:          reflect.TypeOf((*benchmarkTaskRunner)(nil)).Elem(),
			DefaultRunnerType:   reflect.TypeOf(&defaultBenchmarkTaskRunner{}),
			ERRunnerType:        reflect.TypeOf((*benchmarkTaskRunnerER)(nil)).Elem(),
			WrapperERRunnerCtor: newWrapperBenchmarkTaskRunnerER,
			DefaultERRunnerType: reflect.TypeOf(&defaultBenchmarkTaskRunnerER{}),
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
		App:       testTaskServerApp(),
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
			Context:    context.Background(),
			TraceValue: baseTrace,
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

func TestServerRunTaskAddsProfileLabels(t *testing.T) {
	ensureRunnerTaskRegistered()
	executor := &_RunnerRecorderExecutor{profileLabels: map[string]string{}}
	server := NewServer(Option{
		App:       testTaskServerApp(),
		ImplTypes: []reflect.Type{reflect.TypeFor[*testRunnerImpl]()},
		Executor:  executor,
	})
	trace := meta.InitialTrace()
	err := server.RunTask(context.Background(), appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		TaskSkelName:    "runner.task",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{"GroupId":9}`,
	})

	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"vine.app":      "test.app",
		"vine.protocol": "task",
		"vine.task":     "runner.task",
		"vine.trigger":  "forGroup",
	}, executor.profileLabels)
}

func TestServerReturnsInvalidTaskWhenTriggerNotRegistered(t *testing.T) {
	ensureRunnerTaskRegistered()

	logPath := filepath.Join(t.TempDir(), "task-rejected.jsonl")
	server := NewServer(Option{
		App:       testTaskServerApp(),
		ImplTypes: []reflect.Type{reflect.TypeOf(&testRunnerImpl{})},
		Executor:  &_RunnerRecorderExecutor{},
		Logger: logger.New("vine:test", logger.WithOption{
			Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: logPath,
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
	if record["msg"] != "task runner handle rejected" || record["level"] != "ERROR" || record["code"] != string(ex.InvalidTask) {
		t.Fatalf("unexpected rejection record: %#v", record)
	}
	if record["taskSkel"] != "missing.task" || record["taskTriggerSkel"] != "forGroup" {
		t.Fatalf("rejection record must preserve main Task field names: %#v", record)
	}
}

func TestServerConvertsRecoveredPanicToInternalError(t *testing.T) {
	ensureRunnerTaskRegistered()

	server := NewServer(Option{
		App:       testTaskServerApp(),
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
			Context:    context.Background(),
			TraceValue: meta.InitialTrace(),
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
		App:       testTaskServerApp(),
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

func BenchmarkServerRunTask(b *testing.B) {
	ensureBenchmarkTaskRegistered()
	trace := meta.InitialTrace()
	server := NewServer(Option{
		App:       testTaskServerApp(),
		ImplTypes: []reflect.Type{reflect.TypeOf(&benchmarkTaskRunnerImpl{})},
		Executor:  NewContainerExecutor(nil, nil),
		Logger: logger.New("vine:benchmark", logger.WithOption{
			Level: logger.LevelError,
		}),
	})
	run := appskeled.TaskRun{
		Metadata: appskeled.TaskRunMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		TaskSkelName:    "benchmark.task",
		TriggerSkelName: "forGroup",
		ArgumentsJson:   `{"groupId":11}`,
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := server.RunTask(context.Background(), run); err != nil {
			b.Fatal(err)
		}
	}
}
