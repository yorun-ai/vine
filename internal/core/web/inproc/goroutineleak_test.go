//go:build goroutineleak

package inproc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"go.yorun.ai/vine/internal/testutil/goroutineleak"
)

func TestGoroutineLeakCanceledWebInprocRoundTripLifecycle(t *testing.T) {
	runCanceledWebInprocRoundTripLifecycle(t)
	goroutineleak.RequireNone(t)
}

func runCanceledWebInprocRoundTripLifecycle(t *testing.T) {
	const endpoint = "web+inproc://goroutineleak"
	started := make(chan struct{})
	release := make(chan struct{})
	returned := make(chan struct{})
	cleanup := Register(endpoint, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
		close(returned)
	}))
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	result := make(chan error, 1)
	go func() {
		_, err := RoundTrip(endpoint, request)
		result <- err
	}()

	<-started
	cancel()
	if err := <-result; err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	close(release)
	<-returned

	// Let the handler goroutine complete its buffered response send. If the
	// response path regresses to an unbuffered or abandoned send, the following
	// goroutineleak profile reports it after this lifecycle function returns.
	for range 10 {
		runtime.Gosched()
	}
}
