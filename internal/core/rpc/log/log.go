package log

import (
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/redact"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type Span struct {
	logger              *logger.Logger
	finishMsg           string
	startedAt           time.Time
	fields              []any
	method              spec.MethodInfo
	muteSuccess         bool
	debugEnabledAtStart bool
	arguments           _PayloadValue
}

const (
	rpcArgumentsField = "rpcArguments"
	rpcResultField    = "rpcResult"
)

type _PayloadValue struct {
	result redact.Result
	err    error
}

func Noop() *Span {
	return &Span{}
}

func StartClientInvoke(log *logger.Logger, trace meta.Trace, method spec.MethodInfo, serverEndpoint string) *Span {
	span := Start(log, "rpc client invoke started", "rpc client invoke finished", trace, method, nil, nil)
	span.Add("serverEndpoint", serverEndpoint)
	if !span.muteSuccess && span.debugEnabledAtStart {
		log.Debug("rpc client invoke started", span.fields...)
	}
	return span
}

func StartServerHandle(log *logger.Logger, trace meta.Trace, method spec.MethodInfo, client meta.App, server meta.App, arguments ...any) *Span {
	span := Start(log, "rpc server handle started", "rpc server handle finished", trace, method, client, server)
	if !span.debugEnabledAtStart {
		return span
	}
	if len(arguments) > 0 && !isInternalTransport(method) {
		span.arguments = renderRpcArguments(method, arguments[0])
	}
	if span.muteSuccess {
		return span
	}
	if len(arguments) > 0 && !isInternalTransport(method) {
		span.fields = appendPayloadFields(span.fields, rpcArgumentsField, span.arguments)
	}
	log.Debug("rpc server handle started", span.fields...)
	span.fields = removePayloadFields(span.fields, rpcArgumentsField)
	return span
}

func Start(
	log *logger.Logger,
	startMsg string,
	finishMsg string,
	trace meta.Trace,
	method spec.MethodInfo,
	client meta.App,
	server meta.App,
) *Span {
	fields := make([]any, 0, 20)
	fields = appendTraceFields(fields, trace)
	fields = appendMethodFields(fields, method)
	fields = appendAppFields(fields, "client", client)
	fields = appendAppFields(fields, "server", server)

	span := &Span{
		logger:              log,
		finishMsg:           finishMsg,
		startedAt:           time.Now(),
		fields:              fields,
		method:              method,
		muteSuccess:         IsSuccessLogMuted(method),
		debugEnabledAtStart: log.Enabled(logger.LevelDebug),
	}
	if !span.muteSuccess && span.debugEnabledAtStart &&
		startMsg != "rpc server handle started" && startMsg != "rpc client invoke started" {
		log.Debug(startMsg, fields...)
	}
	return span
}

func (s *Span) Add(args ...any) {
	if s.logger == nil || len(args) == 0 {
		return
	}
	s.fields = append(s.fields, args...)
}

func (s *Span) Started() bool {
	return s != nil && s.logger != nil
}

func (s *Span) AddApp(prefix string, app meta.App) {
	if s.logger == nil {
		return
	}
	s.fields = appendAppFields(s.fields, prefix, app)
}

func (s *Span) Finish(err ex.Error) {
	s.finish(err, nil, false)
}

func (s *Span) FinishServer(err ex.Error, result any) {
	s.finish(err, result, true)
}

func (s *Span) finish(err ex.Error, result any, server bool) {
	if s.logger == nil {
		return
	}
	code := ex.OK
	if err != nil {
		code = err.Code()
	}
	if code == ex.OK && s.muteSuccess {
		return
	}
	if code != ex.OK && err != nil {
		s.fields = append(s.fields, "error", err.Error())
		if panicValue, panicked := ex.PanicValue(err); panicked {
			s.fields = append(s.fields, "panic", panicValue)
		}
		if stack := ex.Stack(err); stack != "" {
			s.fields = append(s.fields, "stack", stack)
		}
		if server && s.muteSuccess {
			if s.debugEnabledAtStart {
				s.fields = appendPayloadFields(s.fields, rpcArgumentsField, s.arguments)
			} else {
				s.fields = append(s.fields, "rpcArgumentsOmittedReason", "debug_disabled_at_start")
			}
		}
	} else if server && s.logger.Enabled(logger.LevelDebug) && !isInternalTransport(s.method) {
		payload := renderRpcResult(s.method, result)
		s.fields = appendPayloadFields(s.fields, rpcResultField, payload)
	}
	s.fields = append(s.fields,
		"code", string(code),
		"duration", time.Since(s.startedAt),
	)
	logLifecycle(s.logger, s.finishMsg, err, s.fields...)
}

func (s *Span) FinishWithResponse(err ex.Error, response spec.Response) {
	if s.logger == nil {
		return
	}
	if response != nil {
		s.AddApp("server", response.Server())
	}
	s.Finish(err)
}

func ServerRejected(log *logger.Logger, startedAt time.Time, trace meta.Trace, method spec.MethodInfo, client meta.App, server meta.App, err ex.Error, unresolvedSkelNames ...string) {
	if log == nil {
		return
	}
	fields := make([]any, 0, 24)
	fields = appendTraceFields(fields, trace)
	fields = appendMethodFields(fields, method)
	if method == nil {
		if len(unresolvedSkelNames) > 0 && unresolvedSkelNames[0] != "" {
			fields = append(fields, "rpcServiceSkel", unresolvedSkelNames[0])
		}
		if len(unresolvedSkelNames) > 1 && unresolvedSkelNames[1] != "" {
			fields = append(fields, "rpcMethodSkel", unresolvedSkelNames[1])
		}
	}
	fields = appendAppFields(fields, "client", client)
	fields = appendAppFields(fields, "server", server)
	fields = appendErrorFields(fields, err)
	fields = append(fields, "duration", time.Since(startedAt))
	logLifecycle(log, "rpc server request rejected", err, fields...)
}

func ClientRejected(log *logger.Logger, startedAt time.Time, trace meta.Trace, method spec.MethodInfo, serverEndpoint string, arguments any, err ex.Error) {
	if log == nil {
		return
	}
	fields := make([]any, 0, 22)
	fields = appendTraceFields(fields, trace)
	fields = appendMethodFields(fields, method)
	fields = append(fields, "serverEndpoint", serverEndpoint)
	if log.Enabled(logger.LevelDebug) && !isInternalTransport(method) {
		payload := renderRpcArguments(method, arguments)
		fields = appendPayloadFields(fields, rpcArgumentsField, payload)
	}
	fields = appendErrorFields(fields, err)
	fields = append(fields, "duration", time.Since(startedAt))
	logLifecycle(log, "rpc client invoke rejected", err, fields...)
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

func renderRpcArguments(method spec.MethodInfo, value any) _PayloadValue {
	return renderPayload(value, method != nil && method.ArgumentsSensitive())
}

func renderRpcResult(method spec.MethodInfo, value any) _PayloadValue {
	return renderPayload(value, method != nil && method.ResultSensitive())
}

func renderPayload(value any, sensitive bool) _PayloadValue {
	result, err := redact.Render(value, redact.Option{RootSensitive: sensitive})
	return _PayloadValue{result: result, err: err}
}

func appendPayloadFields(fields []any, name string, payload _PayloadValue) []any {
	if payload.err != nil {
		return append(fields, name+"OmittedReason", "redact_failed")
	}
	if payload.result.JSON != "" {
		fields = append(fields, name, payload.result.JSON)
	}
	if payload.result.Redacted {
		fields = append(fields, name+"Redacted", true)
	}
	if payload.result.Truncated {
		fields = append(fields, name+"Truncated", true)
	}
	return fields
}

func removePayloadFields(fields []any, name string) []any {
	result := fields[:0]
	for index := 0; index+1 < len(fields); index += 2 {
		key, _ := fields[index].(string)
		if key == name || key == name+"Redacted" || key == name+"Truncated" || key == name+"OmittedReason" {
			continue
		}
		result = append(result, fields[index], fields[index+1])
	}
	return result
}

func isInternalTransport(method spec.MethodInfo) bool {
	if method == nil || method.Service() == nil {
		return false
	}
	service := method.Service().SkelName()
	return service == "vine.app.TaskService" || service == "vine.app.EventService"
}

func appendTraceFields(fields []any, trace meta.Trace) []any {
	if trace == nil {
		return fields
	}

	fields = append(fields,
		"vrpcId", trace.Id(),
		"vrpcSpan", trace.Span(),
	)
	if trace.ParentSpan() != "" {
		fields = append(fields, "vrpcParentSpan", trace.ParentSpan())
	}
	return fields
}

func appendMethodFields(fields []any, method spec.MethodInfo) []any {
	if method == nil {
		return fields
	}

	fields = append(fields,
		"rpcMethod", method.Name(),
		"rpcMethodSkel", method.SkelName(),
	)
	if service := method.Service(); service != nil {
		fields = append(fields,
			"rpcApi", service.Name()+"."+method.Name(),
			"rpcService", service.Name(),
			"rpcServiceSkel", service.SkelName(),
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
