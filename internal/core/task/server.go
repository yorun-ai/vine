package task

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime/debug"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	tasklog "go.yorun.ai/vine/internal/core/task/log"
	"go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vpre"
)

var taskServerLogger = logger.NewLogger(logger.GlobalOption())

type Option struct {
	App       meta.App
	ImplTypes []reflect.Type
	Executor  Executor
}

type Server struct {
	opt *Option

	implDict *spec.ImplDict
	executor Executor
}

func NewServer(opt Option) *Server {
	server := &Server{
		opt:      &opt,
		executor: opt.Executor,
	}
	server.init()
	return server
}

func (s *Server) init() {
	vpre.Check(len(s.opt.ImplTypes) > 0, "no task impl type found")
	vpre.CheckNotNil(s.executor, "task executor cannot be nil")
	s.implDict = spec.NewImplDict()
	for _, implType := range s.opt.ImplTypes {
		s.implDict.Add(implType)
	}
	s.executor.Init(*s.implDict)
}

func (s *Server) RunTask(ctx context.Context, run appskeled.TaskRun) ex.Error {
	triggerInfo, ok := spec.GetTriggerInfo(run.TaskSkelName, run.TriggerSkelName)
	if !ok {
		return ex.New(ex.InvalidTask, "unknown task trigger "+run.TaskSkelName+"/"+run.TriggerSkelName)
	}

	trace, err := meta.NewTrace(run.Metadata.TraceId, run.Metadata.TraceSpan)
	if err != nil {
		return ex.New(ex.InvalidRequest, err.Error())
	}
	client, err := meta.NewApp(run.Metadata.AppName, run.Metadata.AppVersion, run.Metadata.AppInstanceId.String())
	if err != nil {
		return ex.New(ex.InvalidRequest, err.Error())
	}

	arguments := triggerInfo.NewArguments()
	err = json.Unmarshal([]byte(run.ArgumentsJson), arguments)
	if err != nil {
		return ex.New(ex.InvalidTask, err.Error())
	}
	err = triggerInfo.ValidateArguments(arguments)
	if err != nil {
		return ex.New(ex.InvalidTask, err.Error())
	}
	triggerImpl, getErr := s.implDict.GetTriggerImplByInfo(triggerInfo)
	if getErr != nil {
		return ex.New(ex.InvalidTask, getErr.Error())
	}

	return s.runTask(&spec.RunImpl{
		ContextValue: &spec.ContextImpl{
			ContextImpl: meta.ContextImpl{
				Context:    ctx,
				TraceValue: trace.NewChildTrace(),
			},
			LauncherValue:   client,
			LaunchedAtValue: run.Metadata.LaunchedAt.Time,
		},
		TriggerImplValue: triggerImpl,
		TriggerInfoValue: triggerInfo,
		ArgumentsValue:   arguments,
	})
}

func (s *Server) runTask(taskRun spec.Run) (err ex.Error) {
	logSpan := tasklog.StartRunnerHandle(
		taskServerLogger,
		taskRun.Context().Trace(),
		taskRun.TriggerInfo(),
		taskRun.Context().Launcher(),
		s.opt.App)

	defer func() { logSpan.Finish(err) }()

	defer func() {
		if reErr := recover(); reErr != nil {
			switch casted := reErr.(type) {
			case ex.Error:
				err = casted
				if casted.Type() == ex.SystemError {
					stack := ex.PanicStack(casted)
					if stack == "" {
						stack = string(debug.Stack())
					}
					fields := []any{
						"error", casted,
						"stack", stack,
						"taskName", taskRun.TriggerInfo().Task().Name(),
						"taskSkelName", taskRun.TriggerInfo().Task().SkelName(),
						"triggerName", taskRun.TriggerInfo().Name(),
						"triggerSkelName", taskRun.TriggerInfo().SkelName(),
						"runnerMethod", taskRun.TriggerInfo().RunnerMethodName(),
					}
					fields = appendPanicAppFields(fields, "launcher", taskRun.Context().Launcher())
					fields = appendPanicAppFields(fields, "runner", s.opt.App)
					taskServerLogger.Error("task server recovered system error", fields...)
				}
			default:
				fields := []any{
					"panic", reErr,
					"stack", string(debug.Stack()),
					"taskName", taskRun.TriggerInfo().Task().Name(),
					"taskSkelName", taskRun.TriggerInfo().Task().SkelName(),
					"triggerName", taskRun.TriggerInfo().Name(),
					"triggerSkelName", taskRun.TriggerInfo().SkelName(),
					"runnerMethod", taskRun.TriggerInfo().RunnerMethodName(),
				}
				fields = appendPanicAppFields(fields, "launcher", taskRun.Context().Launcher())
				fields = appendPanicAppFields(fields, "runner", s.opt.App)
				taskServerLogger.Error("task server recovered panic", fields...)
				err = ex.NewInternal()
			}
		}
		if err != nil && err.Code().IsUnresponsive() {
			err = ex.NewInternal()
		}
	}()
	return s.executor.Execute(taskRun.Context(), taskRun.TriggerImpl(), taskRun.PositionalArguments())
}

func appendPanicAppFields(fields []any, prefix string, app meta.App) []any {
	if app == nil {
		return fields
	}
	return append(fields,
		prefix+"Name", app.Name(),
		prefix+"Version", app.Version(),
		prefix+"InstanceId", app.InstanceId(),
	)
}
