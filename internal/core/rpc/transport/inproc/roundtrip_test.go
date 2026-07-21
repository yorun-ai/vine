package inproc

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type testHandler struct {
	response spec.Response
	calledCh chan struct{}
	waitCh   <-chan struct{}
	called   *atomic.Bool
}

var inprocTestRequestCounter uint64

func (h testHandler) ServeRpc(rpcRequest spec.Request) spec.Response {
	if h.called != nil {
		h.called.Store(true)
	}
	if h.calledCh != nil {
		close(h.calledCh)
	}
	if h.waitCh != nil {
		<-h.waitCh
	}
	return h.response
}

func testRequest(t *testing.T) spec.Request {
	t.Helper()

	trace := meta.InitialTrace()
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	return &spec.RequestImpl{
		ContextValue: context.Background(),
		TraceValue:   trace,
		ClientValue:  client,
		MethodInfoValue: spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
			Name:     "DemoService",
			SkelName: fmt.Sprintf("demo.service.%d", atomic.AddUint64(&inprocTestRequestCounter, 1)),
			Methods: []*spec.MethodSpec{{
				Name:     "Ping",
				SkelName: "ping",
			}},
		}).Methods()[0],
	}
}

func testResponse(t *testing.T) spec.Response {
	t.Helper()

	server, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	return &spec.ResponseImpl{
		ServerValue: server,
		ResultValue: "pong",
		ErrorValue:  ex.NewOK(),
	}
}

func TestRoundTrip(t *testing.T) {
	resetRegistryForTest(t)

	req := testRequest(t)
	want := testResponse(t)
	Register("rpc+inproc://test", testHandler{response: want})

	got, err := RoundTrip("rpc+inproc://test", req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	if got != want {
		t.Fatalf("unexpected response: got %p want %p", got, want)
	}
}

func TestRoundTripRejectsInvalidInput(t *testing.T) {
	resetRegistryForTest(t)

	if _, err := RoundTrip("rpc+inproc://missing", testRequest(t)); err == nil {
		t.Fatalf("expected missing handler error")
	}
	Register("rpc+inproc://nil-response", testHandler{})
	if _, err := RoundTrip("rpc+inproc://nil-response", testRequest(t)); err == nil {
		t.Fatalf("expected nil response error")
	}
}

func TestRoundTripInvokesHandlerInAnotherGoroutine(t *testing.T) {
	resetRegistryForTest(t)

	req := testRequest(t)
	waitCh := make(chan struct{})
	calledCh := make(chan struct{})

	doneCh := make(chan struct{})
	Register("rpc+inproc://async", testHandler{
		response: testResponse(t),
		calledCh: calledCh,
		waitCh:   waitCh,
	})
	go func() {
		defer close(doneCh)
		_, _ = RoundTrip("rpc+inproc://async", req)
	}()

	select {
	case <-calledCh:
	case <-time.After(time.Second):
		t.Fatal("expected handler to be invoked asynchronously")
	}

	select {
	case <-doneCh:
		t.Fatal("expected round trip to wait for handler response")
	default:
	}

	close(waitCh)
	<-doneCh
}

func TestRoundTripReturnsContextError(t *testing.T) {
	resetRegistryForTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &spec.RequestImpl{
		ContextValue: ctx,
		TraceValue:   meta.InitialTrace(),
		MethodInfoValue: spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
			Name:     "DemoService",
			SkelName: fmt.Sprintf("demo.service.%d", atomic.AddUint64(&inprocTestRequestCounter, 1)),
			Methods: []*spec.MethodSpec{{
				Name:     "Ping",
				SkelName: "ping",
			}},
		}).Methods()[0],
	}

	Register("rpc+inproc://canceled", testHandler{
		response: testResponse(t),
		waitCh:   make(chan struct{}),
	})
	_, err := RoundTrip("rpc+inproc://canceled", req)
	if err == nil || err.Code() != ex.InvocationCancelled {
		t.Fatalf("expected InvocationCancelled, got %#v", err)
	}
}

func TestRoundTripDoesNotWaitForHandlerAfterContextCanceled(t *testing.T) {
	resetRegistryForTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	req := testRequest(t).(*spec.RequestImpl)
	req.ContextValue = ctx

	waitCh := make(chan struct{})
	calledCh := make(chan struct{})
	Register("rpc+inproc://cancel-running", testHandler{
		response: testResponse(t),
		calledCh: calledCh,
		waitCh:   waitCh,
	})

	doneCh := make(chan ex.Error, 1)
	go func() {
		_, err := RoundTrip("rpc+inproc://cancel-running", req)
		doneCh <- err
	}()

	select {
	case <-calledCh:
	case <-time.After(time.Second):
		t.Fatal("expected handler to be called")
	}

	cancel()
	select {
	case err := <-doneCh:
		if err == nil || err.Code() != ex.InvocationCancelled {
			t.Fatalf("expected InvocationCancelled, got %#v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected round trip to return without waiting for handler")
	}

	close(waitCh)
}

func TestRoundTripDoesNotInvokeHandlerWhenContextAlreadyCanceled(t *testing.T) {
	resetRegistryForTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := testRequest(t).(*spec.RequestImpl)
	req.ContextValue = ctx

	var called atomic.Bool
	Register("rpc+inproc://already-canceled", testHandler{
		response: testResponse(t),
		called:   &called,
	})

	_, err := RoundTrip("rpc+inproc://already-canceled", req)
	if err == nil || err.Code() != ex.InvocationCancelled {
		t.Fatalf("expected InvocationCancelled, got %#v", err)
	}
	if called.Load() {
		t.Fatal("expected handler not to be called")
	}
}
