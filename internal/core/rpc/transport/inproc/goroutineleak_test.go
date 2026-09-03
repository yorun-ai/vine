//go:build goroutineleak

package inproc

import (
	"context"
	"runtime"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/testutil/goroutineleak"
)

type goroutineLeakHandler struct {
	response spec.Response
	started  chan struct{}
	release  chan struct{}
	returned chan struct{}
}

func (h goroutineLeakHandler) ServeRpc(spec.Request) spec.Response {
	close(h.started)
	<-h.release
	close(h.returned)
	return h.response
}

func TestGoroutineLeakCanceledInprocRoundTripLifecycle(t *testing.T) {
	runCanceledInprocRoundTripLifecycle(t)
	goroutineleak.RequireNone(t)
}

func runCanceledInprocRoundTripLifecycle(t *testing.T) {
	const endpoint = "rpc+inproc://goroutineleak"
	started := make(chan struct{})
	release := make(chan struct{})
	returned := make(chan struct{})
	cleanup := Register(endpoint, goroutineLeakHandler{
		response: testResponse(t),
		started:  started,
		release:  release,
		returned: returned,
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	request := testRequest(t).(*spec.RequestImpl)
	request.ContextValue = ctx
	result := make(chan ex.Error, 1)
	go func() {
		_, err := RoundTrip(endpoint, request)
		result <- err
	}()

	<-started
	cancel()
	if err := <-result; err == nil || err.Code() != ex.InvocationCancelled {
		t.Fatalf("expected InvocationCancelled, got %#v", err)
	}
	close(release)
	<-returned

	// Let the transport goroutine complete its buffered response send. If the
	// response path regresses to an unbuffered or abandoned send, the following
	// goroutineleak profile reports it after this lifecycle function returns.
	for range 10 {
		runtime.Gosched()
	}
}
