package log

import (
	"time"

	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
)

type Span struct {
	logger    *logger.Logger
	finishMsg string
	startedAt time.Time
	fields    []any
}

func Noop() *Span {
	return &Span{}
}

func EmitterEmitSuccess(log *logger.Logger, trace meta.Trace, event spec.EventInfo, emitter meta.App) {
	if log == nil {
		return
	}

	fields := make([]any, 0, 14)
	fields = appendTraceFields(fields, trace)
	fields = appendEventFields(fields, event)
	fields = appendAppFields(fields, "emitter", emitter)
	fields = append(fields, "code", string(ex.OK))
	log.Debug("event emitter emit success", fields...)
}

func StartListenerHandle(log *logger.Logger, trace meta.Trace, event spec.EventInfo, emitter meta.App, listener meta.App) *Span {
	span := Start(log, "event listener handle started", "event listener handle finished", trace, event, emitter)
	span.AddApp("listener", listener)
	return span
}

func Start(log *logger.Logger, startMsg string, finishMsg string, trace meta.Trace, event spec.EventInfo, emitter meta.App) *Span {
	if log == nil {
		return Noop()
	}

	fields := make([]any, 0, 16)
	fields = appendTraceFields(fields, trace)
	fields = appendEventFields(fields, event)
	fields = appendAppFields(fields, "emitter", emitter)

	span := &Span{
		logger:    log,
		finishMsg: finishMsg,
		startedAt: time.Now(),
		fields:    fields,
	}
	log.Debug(startMsg, fields...)
	return span
}

func (s *Span) AddApp(prefix string, app meta.App) {
	if s.logger == nil {
		return
	}
	s.fields = appendAppFields(s.fields, prefix, app)
}

func (s *Span) Finish(err ex.Error) {
	if s.logger == nil {
		return
	}
	code := ex.OK
	if err != nil {
		code = err.Code()
		s.fields = append(s.fields, "error", err.Error())
	}
	s.fields = append(s.fields,
		"code", string(code),
		"duration", time.Since(s.startedAt),
	)
	switch code.Type() {
	case ex.NoError:
		s.logger.Info(s.finishMsg, s.fields...)
	case ex.ApplicationError:
		s.logger.Warn(s.finishMsg, s.fields...)
	default:
		s.logger.Error(s.finishMsg, s.fields...)
	}
}

func appendTraceFields(fields []any, trace meta.Trace) []any {
	if trace == nil {
		return fields
	}

	fields = append(fields,
		"veventId", trace.Id(),
		"veventSpan", trace.Span(),
	)
	if trace.ParentSpan() != "" {
		fields = append(fields, "veventParentSpan", trace.ParentSpan())
	}
	return fields
}

func appendEventFields(fields []any, event spec.EventInfo) []any {
	if event == nil {
		return fields
	}

	return append(fields,
		"eventName", event.Name(),
		"eventSkel", event.SkelName(),
		"eventEmitterMethod", event.EmitterMethodName(),
		"eventListenerMethod", event.ListenerMethodName(),
	)
}

func appendAppFields(fields []any, prefix string, app meta.App) []any {
	if app == nil {
		return fields
	}

	return append(fields,
		prefix+"Name", app.Name(),
		prefix+"Version", app.Version(),
		prefix+"InstanceId", app.InstanceId(),
	)
}
