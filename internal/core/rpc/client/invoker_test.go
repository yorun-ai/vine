package client

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

var invokerTestMethodCounter uint64

type invokerClientResult struct {
	Name string `json:"name"`
}

type invokerServerResult struct {
	Name string `json:"name"`
}

func testMethodInfo() spec.MethodInfo {
	return newInvokerTestMethodInfo("Ping", "ping", nil, nil, nil)
}

func newInvokerTestMethodInfo(name string, skelName string, argumentsType reflect.Type, resultType reflect.Type, validateResult func(any) error) spec.MethodInfo {
	return spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "InvokerTestService",
		SkelName: fmt.Sprintf("invoker.test.service.%s.%d", skelName, atomic.AddUint64(&invokerTestMethodCounter, 1)),
		Methods: []*spec.MethodSpec{{
			Name:           name,
			SkelName:       skelName,
			ArgumentsType:  argumentsType,
			ResultType:     resultType,
			ValidateResult: validateResult,
		}},
	}).Methods()[0]
}

func testClientApp(t *testing.T) meta.App {
	t.Helper()
	app, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	return app
}

func TestInvokerBuildRequestCreatesClientSpan(t *testing.T) {
	trace := meta.InitialTrace()
	if trace == nil {
		t.Fatalf("expected initial trace")
	}

	rpcContext := &meta.ContextImpl{
		Context:    context.Background(),
		TraceValue: trace,
	}
	client := New(Option{Context: rpcContext, Logger: testClientLogger()})
	methodInfo := testMethodInfo()

	first := client.newInvoker(methodInfo, nil, nil).buildRequest()
	second := client.newInvoker(methodInfo, nil, nil).buildRequest()

	if first.Trace().Id() != trace.Id() {
		t.Fatalf("unexpected first trace id: got %s want %s", first.Trace().Id(), trace.Id())
	}
	if first.Trace().ParentSpan() != trace.Span() {
		t.Fatalf("unexpected first parent span: got %s want %s", first.Trace().ParentSpan(), trace.Span())
	}
	if first.Trace().Span() == trace.Span() {
		t.Fatalf("expected first client span to differ from base span")
	}
	if second.Trace().Id() != trace.Id() {
		t.Fatalf("unexpected second trace id: got %s want %s", second.Trace().Id(), trace.Id())
	}
	if second.Trace().ParentSpan() != trace.Span() {
		t.Fatalf("unexpected second parent span: got %s want %s", second.Trace().ParentSpan(), trace.Span())
	}
	if second.Trace().Span() == trace.Span() || second.Trace().Span() == first.Trace().Span() {
		t.Fatalf("expected each request to get a distinct client span")
	}
}

func TestBuildRequestAcceptsMetaContextWithoutRPCClient(t *testing.T) {
	client := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	})

	req := client.newInvoker(testMethodInfo(), nil, nil).buildRequest()
	if req.Context() == nil {
		t.Fatal("expected request context")
	}
	if req.Trace() == nil {
		t.Fatal("expected request trace")
	}
	if req.Client() != nil {
		t.Fatal("expected nil rpc client app for meta-only context")
	}
}

func TestBuildRequestUsesConfiguredApplication(t *testing.T) {
	client := New(Option{
		Context:   testClientContext(),
		ClientApp: testClientApp(t),
		Logger:    testClientLogger(),
	})

	req := client.newInvoker(testMethodInfo(), nil, nil).buildRequest()
	if req.Client() != client.clientApp {
		t.Fatal("expected configured application to be written into rpc request")
	}
}

func TestWithContextAcceptsPlainContextAndKeepsRPCMetadata(t *testing.T) {
	baseTrace := meta.InitialTrace()
	client := New(Option{
		Context: &meta.ContextImpl{
			Context:    context.Background(),
			TraceValue: baseTrace,
		},
		Logger: testClientLogger(),
	})

	parent := context.WithValue(context.Background(), "k", "v")
	req := client.newInvoker(testMethodInfo(), nil, []InvokeOption{
		WithContext(parent),
	}).buildRequest()

	if req.Context().Value("k") != "v" {
		t.Fatal("expected plain context to be used as request parent")
	}
	if req.Trace().Id() != baseTrace.Id() || req.Trace().ParentSpan() != baseTrace.Span() {
		t.Fatal("expected rpc metadata to derive from client default context")
	}
	if _, ok := req.Context().Deadline(); ok {
		t.Fatal("expected WithContext to bypass default timeout")
	}
}

func TestWithContextMetaContextDoesNotOverrideRPCMetadata(t *testing.T) {
	baseTrace := meta.InitialTrace()
	overrideTrace := meta.InitialTrace()
	client := New(Option{
		Context: &meta.ContextImpl{
			Context:    context.Background(),
			TraceValue: baseTrace,
		},
		Logger: testClientLogger(),
	})

	req := client.newInvoker(testMethodInfo(), nil, []InvokeOption{
		WithContext(&meta.ContextImpl{
			Context:    context.Background(),
			TraceValue: overrideTrace,
		}),
	}).buildRequest()

	if req.Trace().Id() != baseTrace.Id() || req.Trace().ParentSpan() != baseTrace.Span() {
		t.Fatal("expected WithContext to not override rpc metadata")
	}
	if req.Trace().Id() == overrideTrace.Id() {
		t.Fatal("expected override context trace to be ignored")
	}
	if _, ok := req.Context().Deadline(); ok {
		t.Fatal("expected WithContext to bypass default timeout")
	}
}

func TestDefaultInvokeAppliesTimeoutWithoutWithContext(t *testing.T) {
	client := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	})

	req := client.newInvoker(testMethodInfo(), nil, nil).buildRequest()
	if _, ok := req.Context().Deadline(); !ok {
		t.Fatal("expected default timeout when WithContext is not used")
	}
}

func TestWithContextAndWithTimeoutCannotBeUsedTogether(t *testing.T) {
	client := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	})

	assertPanics := func(fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		fn()
	}

	assertPanics(func() {
		_ = client.newInvoker(testMethodInfo(), nil, []InvokeOption{
			WithContext(context.Background()),
			WithTimeout(time.Second),
		})
	})
}

func TestParseResponseReturnsTransportError(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(testMethodInfo(), nil, nil)
	expected := ex.New(ex.InvocationFailed, "boom")

	result, err := invoker.parseResponse(nil, expected)
	if result != nil {
		t.Fatal("expected nil result when transport error exists")
	}
	if !errors.Is(err, expected) && err != expected {
		t.Fatalf("expected transport error passthrough, got %#v", err)
	}
}

func TestParseResponseReturnsRPCError(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(testMethodInfo(), nil, nil)
	expected := ex.New(ex.NotFound, "missing")

	result, err := invoker.parseResponse(&spec.ResponseImpl{
		ErrorValue: expected,
	}, nil)
	if result != nil {
		t.Fatal("expected nil result when rpc response contains error")
	}
	if err != expected {
		t.Fatalf("expected rpc error passthrough, got %#v", err)
	}
}

func TestParseResponseAllowsOKErrorAndReturnsResult(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(testMethodInfo(), nil, nil)

	result, err := invoker.parseResponse(&spec.ResponseImpl{
		ResultValue: "ok",
		ErrorValue:  ex.NewOK(),
	}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %#v", err)
	}
	if result != "ok" {
		t.Fatalf("expected result to pass through, got %#v", result)
	}
}

func TestParseResponseRejectsUnexpectedNilResult(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(newInvokerTestMethodInfo("Ping", "ping", nil, nil, func(value any) error {
		return spec.CheckValueNotNil(value, "result")
	}), nil, nil)

	result, err := invoker.parseResponse(&spec.ResponseImpl{}, nil)
	if result != nil {
		t.Fatal("expected nil result")
	}
	if err == nil || err.Code() != ex.UnexpectedResponse {
		t.Fatalf("expected UnexpectedResponse, got %#v", err)
	}
}

func TestParseResponseReturnsSuccessResult(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(newInvokerTestMethodInfo("Ping", "ping", nil, nil, func(value any) error {
		return spec.CheckValueNotNil(value, "result")
	}), nil, nil)

	result, err := invoker.parseResponse(&spec.ResponseImpl{
		ResultValue: "pong",
	}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %#v", err)
	}
	if result != "pong" {
		t.Fatalf("expected pong result, got %#v", result)
	}
}

func TestParseResponseReturnsResponseResultAsIs(t *testing.T) {
	invoker := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	}).newInvoker(newInvokerTestMethodInfo("Ping", "ping", nil, reflect.TypeOf(invokerClientResult{}), nil), nil, nil)

	result, err := invoker.parseResponse(&spec.ResponseImpl{
		ResultValue: invokerServerResult{Name: "vine"},
	}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %#v", err)
	}

	got, ok := result.(invokerServerResult)
	if !ok {
		t.Fatalf("expected invokerServerResult, got %#v", result)
	}
	if got.Name != "vine" {
		t.Fatalf("unexpected result: %#v", got)
	}
}
