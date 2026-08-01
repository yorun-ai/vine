package rpc_test

import (
	"context"
	"testing"
	"time"

	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/core/rpc"
)

func TestFacadeContextAndExecutors(t *testing.T) {
	app := meta.MustNewApp("demo.user", "1.2.3", "550e8400-e29b-41d4-a716-446655440000")
	trace := meta.InitialTrace()
	actor := meta.NewAnonymousActor()
	ctx := rpc.NewContext(context.Background(), trace, app, nil, actor)

	if ctx.Client() != app || ctx.Trace() != trace || ctx.Actor() != actor {
		t.Fatal("expected Rpc context to retain facade metadata")
	}
	if rpc.NewDefaultExecutor() == nil {
		t.Fatal("expected default executor")
	}
	if rpc.NewContainerExecutor(nil, nil) == nil {
		t.Fatal("expected container executor")
	}

	var invokeOption rpc.InvokeOption = rpc.WithTimeout(time.Second)
	if invokeOption == nil {
		t.Fatal("expected timeout invocation option")
	}
}
