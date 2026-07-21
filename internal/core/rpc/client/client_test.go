package client

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

func testClientContext() meta.Context {
	return &meta.ContextImpl{
		Context:    context.Background(),
		TraceValue: meta.InitialTrace(),
	}
}

func testClientLogger() *logger.Logger {
	return logger.NewLogger(logger.GlobalOption())
}

var testClientMethodCounter uint64

func testClientMethodInfo() spec.MethodInfo {
	return spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "ClientTestService",
		SkelName: fmt.Sprintf("client.test.service.%d", atomic.AddUint64(&testClientMethodCounter, 1)),
		Methods: []*spec.MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}).Methods()[0]
}

func TestNewPanicsWhenContextIsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = New(Option{})
}

func TestNewDefaultsReturnIfSystemErrorToFalse(t *testing.T) {
	client := New(Option{
		Context: testClientContext(),
		Logger:  testClientLogger(),
	})

	if client.returnIfSystemError {
		t.Fatal("expected ReturnIfSystemError to default to false")
	}
}

func TestNewUsesConfiguredReturnIfSystemError(t *testing.T) {
	client := New(Option{
		Context:             testClientContext(),
		Logger:              testClientLogger(),
		ReturnIfSystemError: true,
	})

	if !client.returnIfSystemError {
		t.Fatal("expected ReturnIfSystemError to be enabled on client")
	}
}

func TestInvokePanicsOnSystemErrorByDefault(t *testing.T) {
	client := New(Option{
		Context:        testClientContext(),
		ClientApp:      testClientApp(t),
		Logger:         testClientLogger(),
		ServerEndpoint: "http://127.0.0.1:1",
	})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		err, ok := recovered.(ex.Error)
		if !ok {
			t.Fatalf("expected ex.Error panic, got %T", recovered)
		}
		if err.Type() != ex.SystemError {
			t.Fatalf("unexpected error type: got %s want %s", err.Type(), ex.SystemError)
		}
	}()

	_, _ = client.Invoke(testClientMethodInfo(), nil)
}

func TestInvokeReturnsSystemErrorWhenReturnIfSystemErrorEnabled(t *testing.T) {
	client := New(Option{
		Context:             testClientContext(),
		ClientApp:           testClientApp(t),
		Logger:              testClientLogger(),
		ServerEndpoint:      "http://127.0.0.1:1",
		ReturnIfSystemError: true,
	})

	result, err := client.Invoke(testClientMethodInfo(), nil)
	if result != nil {
		t.Fatal("expected nil result on system error")
	}
	if err == nil {
		t.Fatal("expected system error to be returned")
	}
	if err.Type() != ex.SystemError {
		t.Fatalf("unexpected error type: got %s want %s", err.Type(), ex.SystemError)
	}
}

func TestNewPanicsWhenLoggerIsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = New(Option{
		Context: testClientContext(),
	})
}
