package rpc_test

import (
	"context"
	"testing"

	"go.yorun.ai/vine/core/ex"
	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/core/rpc"
)

var _ func(*rpc.Client, rpc.MethodInfo, any, ...rpc.InvokeOption) (string, ex.Error) = (*rpc.Client).InvokeAs[string]

func TestFacadeContextMetadata(t *testing.T) {
	app := meta.MustNewApp("demo.user", "1.2.3", "550e8400-e29b-41d4-a716-446655440000")
	trace := meta.InitialTrace()
	actor := meta.NewAnonymousActor()
	ctx := rpc.NewContext(context.Background(), trace, app, nil, actor)

	if ctx.Client() != app || ctx.Trace() != trace || ctx.Actor() != actor {
		t.Fatal("expected Rpc context to retain facade metadata")
	}
}
