package event

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime/debug"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	eventlog "go.yorun.ai/vine/internal/core/event/log"
	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vpre"
)

var eventServerLogger = logger.NewLogger(logger.GlobalOption())

type Option struct {
	App               meta.App
	ListenerImplTypes []reflect.Type
	Executor          Executor
}

type Server struct {
	opt *Option

	listenerImplDict *spec.ListenerImplDict
	executor         Executor
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
	vpre.Check(len(s.opt.ListenerImplTypes) > 0, "no message impl type found")
	vpre.CheckNotNil(s.executor, "message executor cannot be nil")
	s.listenerImplDict = spec.NewListenerImplDict()
	for _, listenerImplType := range s.opt.ListenerImplTypes {
		s.listenerImplDict.Add(listenerImplType)
	}
	s.executor.Init(*s.listenerImplDict)
}

func (s *Server) OnEvent(ctx context.Context, on appskeled.EventOn) ex.Error {
	eventInfo, ok := spec.GetEventInfo(on.EventSkelName)
	if !ok {
		return ex.New(ex.InvalidEvent, "unknown event "+on.EventSkelName)
	}

	trace, err := meta.NewTrace(on.Metadata.TraceId, on.Metadata.TraceSpan)
	if err != nil {
		return ex.New(ex.InvalidRequest, err.Error())
	}
	client, err := meta.NewApp(on.Metadata.AppName, on.Metadata.AppVersion, on.Metadata.AppInstanceId.String())
	if err != nil {
		return ex.New(ex.InvalidRequest, err.Error())
	}

	payload := reflect.New(eventInfo.PayloadType()).Interface()
	err = json.Unmarshal([]byte(on.EventJson), payload)
	if err != nil {
		return ex.New(ex.InvalidEvent, err.Error())
	}
	listenerImpl, getErr := s.listenerImplDict.GetListenerImplByInfo(eventInfo)
	if getErr != nil {
		return ex.New(ex.InvalidEvent, getErr.Error())
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

func (s *Server) onEvent(messageOn spec.On) (err ex.Error) {
	logSpan := eventlog.StartListenerHandle(
		eventServerLogger,
		messageOn.Context().Trace(),
		messageOn.EventInfo(),
		messageOn.Context().Emitter(),
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
						"eventName", messageOn.EventInfo().Name(),
						"eventSkelName", messageOn.EventInfo().SkelName(),
						"listenerMethod", messageOn.EventInfo().ListenerMethodName(),
					}
					fields = appendPanicAppFields(fields, "emitter", messageOn.Context().Emitter())
					fields = appendPanicAppFields(fields, "listener", s.opt.App)
					eventServerLogger.Error("event server recovered system error", fields...)
				}
			default:
				fields := []any{
					"panic", reErr,
					"stack", string(debug.Stack()),
					"eventName", messageOn.EventInfo().Name(),
					"eventSkelName", messageOn.EventInfo().SkelName(),
					"listenerMethod", messageOn.EventInfo().ListenerMethodName(),
				}
				fields = appendPanicAppFields(fields, "emitter", messageOn.Context().Emitter())
				fields = appendPanicAppFields(fields, "listener", s.opt.App)
				eventServerLogger.Error("event server recovered panic", fields...)
				err = ex.NewInternal()
			}
		}
		if err != nil && err.Code().IsUnresponsive() {
			err = ex.NewInternal()
		}
	}()
	return s.executor.Execute(messageOn.Context(), messageOn.ListenerImpl(), messageOn.EventPayload())
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
