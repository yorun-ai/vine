package event

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	eventlog "go.yorun.ai/vine/internal/core/event/log"
	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vpre"
)

type Option struct {
	App meta.App
	// LogicalAppName is the stable ApplicationSpec name used for scoped lifecycle logging.
	LogicalAppName string
	// Logger overrides dynamic App and event lifecycle logging when non-nil.
	Logger            *logger.Logger
	ListenerImplTypes []reflect.Type
	Executor          Executor
}

type Server struct {
	opt *Option

	listenerImplDict *spec.ListenerImplDict
	executor         Executor
	log              *logger.Logger
}

func NewServer(opt Option) *Server {
	logger.FreezePayloadPolicies()
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
		server.log = logger.NewScopedLogger(logger.Scope{AppName: appName, Subsystem: logger.SubsystemEvent})
	}
	server.init()
	return server
}

func (s *Server) init() {
	vpre.Check(len(s.opt.ListenerImplTypes) > 0, "no message impl type found")
	vpre.CheckNotNil(s.executor, "message executor cannot be nil")
	s.listenerImplDict = spec.NewListenerImplDict()
	for _, listenerImplType := range s.opt.ListenerImplTypes {
		s.listenerImplDict.Add(listenerImplType)
	}
	s.executor.Init(*s.listenerImplDict)
}

func (s *Server) OnEvent(ctx context.Context, on appskeled.EventOn) ex.Error {
	startedAt := time.Now()
	eventInfo, ok := spec.GetEventInfo(on.EventSkelName)
	if !ok {
		err := ex.New(ex.InvalidEvent, "unknown event "+on.EventSkelName)
		eventlog.ListenerRejected(s.log, startedAt, nil, nil, nil, s.opt.App, err, on.EventSkelName)
		return err
	}

	trace, err := meta.NewTrace(on.Metadata.TraceId, on.Metadata.TraceSpan)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		eventlog.ListenerRejected(s.log, startedAt, nil, eventInfo, nil, s.opt.App, rejectedErr)
		return rejectedErr
	}
	client, err := meta.NewApp(on.Metadata.AppName, on.Metadata.AppVersion, on.Metadata.AppInstanceId.String())
	if err != nil {
		rejectedErr := ex.New(ex.InvalidRequest, err.Error())
		eventlog.ListenerRejected(s.log, startedAt, trace, eventInfo, nil, s.opt.App, rejectedErr)
		return rejectedErr
	}

	payload := reflect.New(eventInfo.PayloadType()).Interface()
	err = json.Unmarshal([]byte(on.EventJson), payload)
	if err != nil {
		rejectedErr := ex.New(ex.InvalidEvent, err.Error())
		eventlog.ListenerRejected(s.log, startedAt, trace, eventInfo, client, s.opt.App, rejectedErr)
		return rejectedErr
	}
	listenerImpl, getErr := s.listenerImplDict.GetListenerImplByInfo(eventInfo)
	if getErr != nil {
		rejectedErr := ex.New(ex.InvalidEvent, getErr.Error())
		eventlog.ListenerRejected(s.log, startedAt, trace, eventInfo, client, s.opt.App, rejectedErr)
		return rejectedErr
	}

	return s.onEvent(&spec.OnImpl{
		ContextValue: &spec.ContextImpl{
			ContextImpl: meta.ContextImpl{
				Context:    ctx,
				TraceValue: trace.NewChildTrace(),
			},
			EmitterValue:   client,
			EmittedAtValue: on.Metadata.EmittedAt.Time,
		},
		EventInfoValue:    eventInfo,
		ListenerImplValue: listenerImpl,
		EventPayloadValue: payload,
	})
}

func (s *Server) onEvent(messageOn spec.On) (responseErr ex.Error) {
	var logErr ex.Error
	logSpan := eventlog.StartListenerHandle(
		s.log,
		messageOn.Context().Trace(),
		messageOn.EventInfo(),
		messageOn.Context().Emitter(),
		s.opt.App,
		messageOn.EventPayload())

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
	logErr = s.executor.Execute(messageOn.Context(), messageOn.ListenerImpl(), messageOn.EventPayload())
	return logErr
}
