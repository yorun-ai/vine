package log

import (
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type Span struct {
	logger      *logger.Logger
	finishMsg   string
	startedAt   time.Time
	fields      []any
	muteSuccess bool
}

func Noop() *Span {
	return &Span{}
}

func StartClientInvoke(log *logger.Logger, trace meta.Trace, method spec.MethodInfo, serverEndpoint string) *Span {
	span := Start(log, "rpc client invoke started", "rpc client invoke finished", trace, method, nil, nil)
	span.Add("serverEndpoint", serverEndpoint)
	return span
}

func StartServerHandle(log *logger.Logger, trace meta.Trace, method spec.MethodInfo, client meta.App, server meta.App) *Span {
	return Start(log, "rpc server handle started", "rpc server handle finished", trace, method, client, server)
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
		logger:      log,
		finishMsg:   finishMsg,
		startedAt:   time.Now(),
		fields:      fields,
		muteSuccess: IsSuccessLogMuted(method),
	}
	if !span.muteSuccess {
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
	if code == ex.OK && s.muteSuccess {
		return
	}
	if err != nil {
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

func (s *Span) FinishWithResponse(err ex.Error, response spec.Response) {
	if s.logger == nil {
		return
	}
	if response != nil {
		s.AddApp("server", response.Server())
	}
	s.Finish(err)
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
