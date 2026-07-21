package app

import (
	"log/slog"

	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

func buildLoggerFields(ctx meta.Context, method rpcspec.MethodInfo, info meta.App) []slog.Attr {
	fields := buildContextLogFields(ctx)

	if method != nil && method.Service() != nil {
		fields = append(fields,
			slog.String("rpcApi", method.Service().Name()+"."+method.Name()),
			slog.String("rpcService", method.Service().Name()),
			slog.String("rpcMethod", method.Name()),
		)
	}

	if info != nil {
		fields = append(fields,
			slog.String("name", info.Name()),
			slog.String("version", info.Version()),
			slog.String("instanceId", info.InstanceId()),
		)
	}

	return fields
}

func buildContextLogFields(ctx meta.Context) []slog.Attr {
	fields := make([]slog.Attr, 0, 7)
	if ctx == nil {
		return fields
	}
	if trace := ctx.Trace(); trace != nil {
		fields = append(fields,
			slog.String("traceId", trace.Id()),
			slog.String("traceSpan", trace.Span()),
		)
	}
	if rpcContext, ok := ctx.(rpcspec.Context); ok {
		if client := rpcContext.Client(); client != nil {
			fields = append(fields,
				slog.String("clientVersion", client.Version()),
				slog.String("clientInstanceId", client.InstanceId()),
				slog.String("clientName", client.Name()),
			)
		}
	}
	if actor := ctx.Actor(); actor != nil {
		fields = append(fields, slog.String("actorType", string(actor.Type())))
	}
	if initiator := ctx.Initiator(); initiator != nil && initiator.IpAddr() != "" {
		fields = append(fields, slog.String("clientIp", initiator.IpAddr()))
	}
	return fields
}
