package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type clientEncodingArguments struct {
	Unsupported chan int `json:"unsupported" arg:"0"`
}

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

func TestInvokeEncodingFailureLogsRejectedWithoutStarted(t *testing.T) {
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "EncodingService",
		SkelName: "client.test.encoding",
		Methods: []*spec.MethodSpec{{
			Name:          "Encode",
			SkelName:      "encode",
			ArgumentsType: reflect.TypeFor[clientEncodingArguments](),
		}},
	}).Methods()[0]
	logPath := filepath.Join(t.TempDir(), "client-rejected.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: logPath})
	client := New(Option{
		Context:             testClientContext(),
		ClientApp:           testClientApp(t),
		Logger:              log,
		ServerEndpoint:      "http://127.0.0.1:1",
		ReturnIfSystemError: true,
	})

	_, err := client.Invoke(method, &clientEncodingArguments{Unsupported: make(chan int)})
	if err == nil || err.Code() != ex.InvocationFailed {
		t.Fatalf("unexpected encoding error: %#v", err)
	}
	logBytes, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read client rejection log: %v", readErr)
	}
	lines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")
	if len(lines) != 1 {
		t.Fatalf("encoding rejection should emit exactly one record: %s", logBytes)
	}
	var record map[string]any
	if decodeErr := json.Unmarshal([]byte(lines[0]), &record); decodeErr != nil {
		t.Fatalf("decode client rejection log: %v", decodeErr)
	}
	if record["msg"] != "rpc client invoke rejected" || record["level"] != "DEBUG" || record["code"] != string(ex.InvocationFailed) {
		t.Fatalf("unexpected client rejection record: %#v", record)
	}
	if _, started := record["phase"]; started {
		t.Fatalf("lifecycle phase field must not be emitted: %#v", record)
	}
	stack, _ := record["stack"].(string)
	arguments, _ := record["rpcArguments"].(string)
	if stack == "" || arguments == "" {
		t.Fatalf("local rejection should include stack and safe arguments: %#v", record)
	}
}
