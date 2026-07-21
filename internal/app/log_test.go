package app

import (
	"context"
	"log/slog"
	"net/netip"
	"testing"

	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
)

func TestBuildContextLogFieldsHandlesNilValues(t *testing.T) {
	fields := buildContextLogFields(nil)
	if len(fields) != 0 {
		t.Fatalf("expected no fields, got %v", fields)
	}
}

func TestBuildContextLogFieldsIncludesAvailableValues(t *testing.T) {
	trace, err := meta.NewTrace(meta.NewId(), meta.NewSpan())
	if err != nil {
		t.Fatalf("unexpected trace error: %v", err)
	}
	actor := meta.NewAuthenticatedActorForTest()
	if err != nil {
		t.Fatalf("unexpected actor error: %v", err)
	}
	client, err := meta.NewApp("svc", "1.2.3", "00000000-0000-0000-0000-000000000123")
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	initiator, err := meta.NewInitiator("init", "1.0.0", "00000000-0000-0000-0000-000000000456", "http", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected initiator error: %v", err)
	}

	fields := buildContextLogFields(rpcspec.NewContext(context.Background(), trace, client, initiator, actor))
	fieldMap := toFieldMap(fields)

	if fieldMap["traceId"] != trace.Id() || fieldMap["traceSpan"] != trace.Span() {
		t.Fatalf("unexpected trace fields: %v", fieldMap)
	}
	if fieldMap["clientName"] != "svc" || fieldMap["clientVersion"] != "1.2.3" || fieldMap["clientInstanceId"] != "00000000-0000-0000-0000-000000000123" {
		t.Fatalf("unexpected client fields: %v", fieldMap)
	}
	if fieldMap["actorType"] != string(meta.ActorTypeAuthenticated) {
		t.Fatalf("unexpected actor field: %v", fieldMap)
	}
	if fieldMap["clientIp"] != netip.MustParseAddr("127.0.0.1").String() {
		t.Fatalf("unexpected initiator field: %v", fieldMap)
	}
}

func toFieldMap(fields []slog.Attr) map[string]any {
	out := make(map[string]any, len(fields))
	for _, attr := range fields {
		switch attr.Value.Kind() {
		case slog.KindString:
			out[attr.Key] = attr.Value.String()
		case slog.KindInt64:
			out[attr.Key] = int(attr.Value.Int64())
		default:
			out[attr.Key] = attr.Value.Any()
		}
	}
	return out
}
