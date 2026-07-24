package log

import (
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/task/spec"
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

func LauncherLaunchSuccess(log *logger.Logger, trace meta.Trace, trigger spec.TriggerInfo, launcher meta.App) {
	if log == nil {
		return
	}

	fields := make([]any, 0, 16)
	fields = appendTraceFields(fields, trace)
	fields = appendTaskFields(fields, trigger)
	fields = appendAppFields(fields, "launcher", launcher)
	fields = append(fields, "code", string(ex.OK))
	log.Debug("task launcher launch success", fields...)
}

func StartRunnerHandle(log *logger.Logger, trace meta.Trace, trigger spec.TriggerInfo, launcher meta.App, runner meta.App) *Span {
	span := Start(log, "task runner handle started", "task runner handle finished", trace, trigger, launcher)
	span.AddApp("runner", runner)
	if log != nil {
		log.Debug("task runner handle started", span.fields...)
	}
	return span
}

func Start(log *logger.Logger, startMsg string, finishMsg string, trace meta.Trace, trigger spec.TriggerInfo, launcher meta.App) *Span {
	if log == nil {
		return Noop()
	}

	fields := make([]any, 0, 18)
	fields = appendTraceFields(fields, trace)
	fields = appendTaskFields(fields, trigger)
	fields = appendAppFields(fields, "launcher", launcher)

	span := &Span{
		logger:    log,
		finishMsg: finishMsg,
		startedAt: time.Now(),
		fields:    fields,
	}
	if startMsg != "task runner handle started" {
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
	logLifecycle(s.logger, s.finishMsg, err, s.fields...)
}

func RunnerRejected(log *logger.Logger, startedAt time.Time, trace meta.Trace, trigger spec.TriggerInfo, launcher meta.App, runner meta.App, err ex.Error, unresolvedSkelNames ...string) {
	if log == nil {
		return
	}
	fields := make([]any, 0, 22)
	fields = appendTraceFields(fields, trace)
	fields = appendTaskFields(fields, trigger)
	if trigger == nil {
		if len(unresolvedSkelNames) > 0 && unresolvedSkelNames[0] != "" {
			fields = append(fields, "taskSkel", unresolvedSkelNames[0])
		}
		if len(unresolvedSkelNames) > 1 && unresolvedSkelNames[1] != "" {
			fields = append(fields, "taskTriggerSkel", unresolvedSkelNames[1])
		}
	}
	fields = appendAppFields(fields, "launcher", launcher)
	fields = appendAppFields(fields, "runner", runner)
	fields = appendErrorFields(fields, err)
	fields = append(fields, "duration", time.Since(startedAt))
	logLifecycle(log, "task runner handle rejected", err, fields...)
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

func logLifecycle(log *logger.Logger, msg string, err ex.Error, fields ...any) {
	if _, panicked := ex.PanicValue(err); panicked {
		log.Error(msg, fields...)
		return
	}

	code := ex.OK
	if err != nil {
		code = err.Code()
	}
	switch code.Type() {
	case ex.SystemError:
		log.Error(msg, fields...)
	case ex.ApplicationError:
		log.Info(msg, fields...)
	default:
		log.Debug(msg, fields...)
	}
}

func appendTraceFields(fields []any, trace meta.Trace) []any {
	if trace == nil {
		return fields
	}

	fields = append(fields,
		"vtaskId", trace.Id(),
		"vtaskSpan", trace.Span(),
	)
	if trace.ParentSpan() != "" {
		fields = append(fields, "vtaskParentSpan", trace.ParentSpan())
	}
	return fields
}

func appendTaskFields(fields []any, trigger spec.TriggerInfo) []any {
	if trigger == nil {
		return fields
	}

	fields = append(fields,
		"taskTriggerName", trigger.Name(),
		"taskTriggerSkel", trigger.SkelName(),
		"taskLauncherMethod", trigger.LauncherMethodName(),
		"taskRunnerMethod", trigger.RunnerMethodName(),
	)
	if task := trigger.Task(); task != nil {
		fields = append(fields,
			"taskName", task.Name(),
			"taskSkel", task.SkelName(),
		)
	}
	return fields
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
