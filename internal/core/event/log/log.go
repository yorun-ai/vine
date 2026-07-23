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

func StartListenerHandle(log *logger.Logger, trace meta.Trace, event spec.EventInfo, emitter meta.App, listener meta.App, payload ...any) *Span {
	span := Start(log, "event listener handle started", "event listener handle finished", trace, event, emitter)
	span.AddApp("listener", listener)
	if len(payload) > 0 && log != nil && log.Enabled(logger.LevelDebug) {
		value := logger.RenderPayload(logger.PayloadDescriptor{
			Surface:       logger.PayloadSurfaceEvent,
			EventSkelName: event.SkelName(),
		}, payload[0])
		span.fields = appendPayloadFields(span.fields, value)
		log.Debug("event listener handle started", span.fields...)
		span.fields = removePayloadFields(span.fields)
	}
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
	if startMsg != "event listener handle started" {
		log.Debug(startMsg, fields...)
	}
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
	}
	if code != ex.OK && err != nil {
		s.fields = append(s.fields, "error", err.Error())
		if panicValue, panicked := ex.PanicValue(err); panicked {
			s.fields = append(s.fields, "panic", panicValue)
		}
		if stack := ex.Stack(err); stack != "" {
			s.fields = append(s.fields, "stack", stack)
		}
	}
	s.fields = append(s.fields,
		"code", string(code),
		"duration", time.Since(s.startedAt),
	)
	logLifecycle(s.logger, s.finishMsg, s.fields...)
}

func ListenerRejected(log *logger.Logger, startedAt time.Time, trace meta.Trace, event spec.EventInfo, emitter meta.App, listener meta.App, err ex.Error, unresolvedEventSkelNames ...string) {
	if log == nil {
		return
	}
	fields := make([]any, 0, 20)
	fields = appendTraceFields(fields, trace)
	fields = appendEventFields(fields, event)
	if event == nil && len(unresolvedEventSkelNames) > 0 && unresolvedEventSkelNames[0] != "" {
		fields = append(fields, "eventSkel", unresolvedEventSkelNames[0])
	}
	fields = appendAppFields(fields, "emitter", emitter)
	fields = appendAppFields(fields, "listener", listener)
	fields = appendErrorFields(fields, err)
	fields = append(fields, "duration", time.Since(startedAt))
	logLifecycle(log, "event listener handle rejected", fields...)
}

func appendPayloadFields(fields []any, payload logger.PayloadValue) []any {
	if payload.JSON != "" {
		fields = append(fields, "eventPayload", payload.JSON)
	}
	if payload.Redacted {
		fields = append(fields, "eventPayloadRedacted", true)
	}
	if payload.OmittedReason != "" && payload.OmittedReason != "policy_off" {
		fields = append(fields, "eventPayloadOmittedReason", payload.OmittedReason)
	}
	return fields
}

func removePayloadFields(fields []any) []any {
	result := fields[:0]
	for index := 0; index+1 < len(fields); index += 2 {
		key, _ := fields[index].(string)
		if key == "eventPayload" || key == "eventPayloadRedacted" || key == "eventPayloadOmittedReason" {
			continue
		}
		result = append(result, fields[index], fields[index+1])
	}
	return result
}

func appendErrorFields(fields []any, err ex.Error) []any {
	code := ex.OK
	if err != nil {
		code = err.Code()
	}
	fields = append(fields, "code", string(code))
	if code == ex.OK || err == nil {
		return fields
	}
	fields = append(fields, "error", err.Error())
	if panicValue, panicked := ex.PanicValue(err); panicked {
		fields = append(fields, "panic", panicValue)
	}
	if stack := ex.Stack(err); stack != "" {
		fields = append(fields, "stack", stack)
	}
	return fields
}

func logLifecycle(log *logger.Logger, msg string, fields ...any) {
	log.Debug(msg, fields...)
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
