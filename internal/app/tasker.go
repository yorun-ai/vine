package app

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/ex"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcserver "go.yorun.ai/vine/internal/core/rpc/server"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/task"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vpre"
)

type TaskerSpec interface {
	TaskerBind(b *di.Binder)
	TaskerInitRunners(addRunner RunnerTypeAdder)
	TaskerInitFilters(addFilter TypeAdder)
}

type TaskerEnabled struct{}

func (*TaskerEnabled) TaskerBind(*di.Binder)             {}
func (*TaskerEnabled) TaskerInitRunners(RunnerTypeAdder) {}
func (*TaskerEnabled) TaskerInitFilters(TypeAdder)       {}

type _Tasker struct {
	spec        TaskerSpec
	appInfo     runtime.App
	bindAppDeps di.BindApplier

	runners    []_RunnerTypeEntry
	taskServer *task.Server
	rpcServer  *rpcserver.Server
}

func newTasker(spec TaskerSpec, info runtime.App, deps di.BindApplier) *_Tasker {
	t := &_Tasker{
		spec:        spec,
		appInfo:     info,
		bindAppDeps: deps,
	}
	t.init()
	return t
}

func (t *_Tasker) init() {
	bindAppliers := []di.BindApplier{
		t.bindAppDeps,
		t.bindContext,
		t.bindLogger,
		t.spec.TaskerBind,
	}

	t.runners = t.collectRunners()
	t.taskServer = task.NewServer(task.Option{
		App:       t.appInfo,
		ImplTypes: t.runnerTypes(),
		Executor:  task.NewContainerExecutor(t.filterTypes(), bindAppliers),
	})

	t.rpcServer = rpcserver.New(rpcserver.Option{
		App:          t.appInfo,
		HandlerTypes: []reflect.Type{T[*_AppTaskServiceServerImpl]()},
		Executor:     rpcserver.NewDefaultExecutor(rpcserver.With(t.taskServer)),
	})
}

func (t *_Tasker) runnerTypes() []reflect.Type {
	var runnerTypes []reflect.Type
	for _, runnerEntry := range t.runners {
		runnerTypes = append(runnerTypes, runnerEntry.kind)
	}
	return runnerTypes
}

func (t *_Tasker) collectRunners() []_RunnerTypeEntry {
	var runnerEntries []_RunnerTypeEntry
	t.spec.TaskerInitRunners(func(runnerType reflect.Type, options ...RunnerOption) {
		runnerEntries = append(runnerEntries, _RunnerTypeEntry{
			kind:    runnerType,
			options: newRunnerOptions(options),
		})
	})
	return runnerEntries
}

func (t *_Tasker) filterTypes() []reflect.Type {
	var filterTypes []reflect.Type
	t.spec.TaskerInitFilters(func(filterType reflect.Type) {
		filterTypes = append(filterTypes, filterType)
	})
	return filterTypes
}

func (*_Tasker) bindContext(b *di.Binder) {
	b.BindFactory(func(ctx taskspec.Context) context.Context {
		return ctx
	})
	b.BindFactory(func(ctx taskspec.Context) meta.Context {
		return ctx
	})
}

func (t *_Tasker) bindLogger(b *di.Binder) {
	b.BindFactory(func(ctx taskspec.Context, triggerInfo taskspec.TriggerInfo) *logger.Logger {
		fields := buildContextLogFields(ctx)
		if triggerInfo != nil {
			fields = append(fields,
				slog.String("taskName", triggerInfo.Task().Name()),
				slog.String("taskSkelName", triggerInfo.Task().SkelName()),
				slog.String("taskTrigger", triggerInfo.Name()),
				slog.String("taskTriggerSkelName", triggerInfo.SkelName()),
			)
		}
		if t.appInfo != nil {
			fields = append(fields,
				slog.String("name", t.appInfo.Name()),
				slog.String("version", t.appInfo.Version()),
				slog.String("instanceId", t.appInfo.InstanceId()),
			)
		}
		return logger.NewLogger(logger.GlobalOption()).With(fields...)
	})
}

func (t *_Tasker) httpHandler() http.Handler {
	return t.rpcServer.HTTPHandler()
}

func (t *_Tasker) rpcHandler() rpcspec.RpcHandler {
	return t.rpcServer.RpcHandler()
}

func (t *_Tasker) taskRunnerRegistrations() []linkskeled.TaskRunnerRegistration {
	implDict := taskspec.NewImplDict()
	infoByType := map[reflect.Type]taskspec.TaskInfo{}
	for _, runnerEntry := range t.runners {
		implDict.Add(runnerEntry.kind)
	}
	implDict.IterateTaskImpl(func(taskImpl taskspec.TaskImpl) {
		infoByType[taskImpl.Type()] = taskImpl.Info()
	})

	registrations := make([]linkskeled.TaskRunnerRegistration, 0, len(t.runners))
	for _, runnerEntry := range t.runners {
		taskInfo := infoByType[runnerEntry.kind]
		registrations = append(registrations, linkskeled.TaskRunnerRegistration{
			TaskSkelName:   taskInfo.SkelName(),
			SchemaHash:     taskInfo.Hash(),
			TimeoutMs:      int(runnerEntry.options.Timeout / time.Millisecond),
			Concurrency:    runnerEntry.options.Concurrency,
			NoRetry:        runnerEntry.options.NoRetry,
			CronSchedulers: toLinkTaskRunnerCronSchedulers(taskInfo, runnerEntry.options.CronSchedulers),
		})
	}
	return registrations
}

func toLinkTaskRunnerCronSchedulers(taskInfo taskspec.TaskInfo, schedulers []RunnerCronScheduler) []linkskeled.TaskRunnerCronScheduler {
	ret := make([]linkskeled.TaskRunnerCronScheduler, 0, len(schedulers))
	for _, scheduler := range schedulers {
		triggerInfo := findTaskTriggerInfoBySkelName(taskInfo, scheduler.TriggerSkelName)
		vpre.CheckNotNil(triggerInfo, "runner cron scheduler trigger %s not found on task %s", scheduler.TriggerSkelName, taskInfo.SkelName())
		vpre.Check(!triggerInfo.HasArguments(), "runner cron scheduler trigger %s must have no arguments", scheduler.TriggerSkelName)
		ret = append(ret, linkskeled.TaskRunnerCronScheduler{
			TriggerSkelName: triggerInfo.SkelName(),
			CronExpr:        scheduler.CronExpr,
		})
	}
	return ret
}

func findTaskTriggerInfoBySkelName(taskInfo taskspec.TaskInfo, triggerSkelName string) taskspec.TriggerInfo {
	for _, triggerInfo := range taskInfo.Triggers() {
		if triggerInfo.SkelName() == triggerSkelName {
			return triggerInfo
		}
	}
	return nil
}

func (*_Tasker) start() {}

// AppTaskServiceServerImpl

type _AppTaskServiceServerImpl struct {
	appskeled.DefaultTaskServiceServer

	Context    rpcspec.Context
	TaskServer *task.Server
}

func (s *_AppTaskServiceServerImpl) RunTask(run appskeled.TaskRun) {
	runErr := s.TaskServer.RunTask(s.Context, run)
	ex.PanicIfError(runErr)
}
