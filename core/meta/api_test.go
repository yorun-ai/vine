package meta_test

import (
	"context"
	"testing"

	"go.yorun.ai/vine/core/meta"
)

const facadeInstanceID = "550e8400-e29b-41d4-a716-446655440000"

func TestFacadeMetadataRoundTrip(t *testing.T) {
	app := meta.MustNewApp("demo.user", "1.2.3", facadeInstanceID)
	decodedApp, err := meta.DecodeAppFromDelimited(meta.EncodeAppToDelimited(app))
	if err != nil {
		t.Fatalf("DecodeAppFromDelimited() error = %v", err)
	}
	if decodedApp.Name() != app.Name() || decodedApp.Version() != app.Version() || decodedApp.InstanceId() != app.InstanceId() {
		t.Fatalf("unexpected decoded app: %s %s %s", decodedApp.Name(), decodedApp.Version(), decodedApp.InstanceId())
	}

	trace := meta.InitialTrace()
	decodedTrace, err := meta.DecodeTraceFromDelimited(meta.EncodeTraceToDelimited(trace))
	if err != nil {
		t.Fatalf("DecodeTraceFromDelimited() error = %v", err)
	}
	actor := meta.NewAnonymousActor()
	ctx := meta.NewContext(context.Background(), decodedTrace, nil, actor)
	if ctx.Trace().Id() != trace.Id() || ctx.Trace().Span() != trace.Span() {
		t.Fatal("expected facade context to retain trace metadata")
	}
	if ctx.Actor().Type() != meta.ActorTypeAnonymous {
		t.Fatalf("unexpected actor type: %s", ctx.Actor().Type())
	}
}
