package task

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	tasklog "go.yorun.ai/vine/internal/core/task/log"
	"go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	App meta.App
	// LogicalAppName is the stable ApplicationSpec name used for scoped lifecycle logging.
	LogicalAppName string
	// Logger overrides dynamic App and task lifecycle logging when non-nil.
	Logger    *logger.Logger
	ImplTypes []reflect.Type
	Executor  Executor
}

type Server struct {
	opt *Option

	implDict *spec.ImplDict
	executor Executor
	log      *logger.Logger
}

func NewServer(opt Option) *Server {
	server := &Server{
		opt:      &opt,
		executor: opt.Executor,
	}
	if opt.Logger != nil {
		server.log = opt.Logger
	} else {
		appName := opt.LogicalAppName
		if appName == "" && opt.App != nil {
			appName = opt.App.Name()
		}
		server.log = logger.NewScopedLogger(logger.Scope{AppName: appName, Subsystem: logger.SubsystemTask})
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
	startedAt := time.Now()
	triggerInfo, ok := spec.GetTriggerInfo(run.TaskSkelName, run.TriggerSkelName)
	if !ok {
		err := ex.New(ex.InvalidTask, "unknown task trigger "+run.TaskSkelName+"/"+run.TriggerSkelName)
		tasklog.RunnerRejected(s.log, startedAt, nil, nil, nil, s.opt.App, err, run.TaskSkelName, run.TriggerSkelName)
		return err
	}

	trace, err := meta.NewTrace(run.Metadata.TraceId, run.Metadata.TraceSpan)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		tasklog.RunnerRejected(s.log, startedAt, nil, triggerInfo, nil, s.opt.App, rejectedErr)
		return rejectedErr
	}
	client, err := meta.NewApp(run.Metadata.AppName, run.Metadata.AppVersion, run.Metadata.AppInstanceId.String())
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		tasklog.RunnerRejected(s.log, startedAt, trace, triggerInfo, nil, s.opt.App, rejectedErr)
		return rejectedErr
	}

	arguments := triggerInfo.NewArguments()
	err = json.Unmarshal([]byte(run.ArgumentsJson), arguments)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidTask, err.Error())
		tasklog.RunnerRejected(s.log, startedAt, trace, triggerInfo, client, s.opt.App, rejectedErr)
		return rejectedErr
	}
	err = triggerInfo.ValidateArguments(arguments)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidTask, err.Error())
		tasklog.RunnerRejected(s.log, startedAt, trace, triggerInfo, client, s.opt.App, rejectedErr)
		return rejectedErr
	}
	triggerImpl, getErr := s.implDict.GetTriggerImplByInfo(triggerInfo)
	if getErr != nil {
		rejectedErr := ex.New(ex.InvalidTask, getErr.Error())
		tasklog.RunnerRejected(s.log, startedAt, trace, triggerInfo, client, s.opt.App, rejectedErr)
		return rejectedErr
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

func (s *Server) runTask(taskRun spec.Run) (responseErr ex.Error) {
	var logErr ex.Error
	logSpan := tasklog.StartRunnerHandle(
		s.log,
		taskRun.Context().Trace(),
		taskRun.TriggerInfo(),
		taskRun.Context().Launcher(),
		s.opt.App)

	defer func() { logSpan.Finish(logErr) }()

	defer func() {
		if reErr := recover(); reErr != nil {
			logErr = ex.RecoverExecution(reErr)
		}
		responseErr = logErr
		if responseErr != nil && responseErr.Code().IsUnresponsive() {
			responseErr = ex.NewInternal()
		}
	}()
	logErr = s.executor.Execute(taskRun.Context(), taskRun.TriggerImpl(), taskRun.PositionalArguments())
	return logErr
}
