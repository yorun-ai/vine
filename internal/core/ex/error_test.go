package ex

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.yorun.ai/vine/util/vcode"
)

func TestErrorTypeDerivedFromCode(t *testing.T) {
	err := New(NotFound, "missing user")

	if got := err.Type(); got != ApplicationError {
		t.Fatalf("unexpected error type: got %s", got)
	}
}

func TestNewInternalUsesCodeDefaultMessage(t *testing.T) {
	err := NewInternal()

	if got := err.Message(); got != Internal.DefaultMessage() {
		t.Fatalf("unexpected internal message: got %q want %q", got, Internal.DefaultMessage())
	}
}

func TestFFormatsMessage(t *testing.T) {
	got := F("field %s is invalid", "email")

	if got != "field email is invalid" {
		t.Fatalf("unexpected formatted message: %q", got)
	}
}

func TestWithCauseSupportsUnwrap(t *testing.T) {
	cause := errors.New("disk offline")
	err := New(OperationFailed, "write failed", WithCause(cause))

	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause to support errors.Is")
	}
}

func TestWithDetailSetsDetailText(t *testing.T) {
	err := New(OperationFailed, "write failed", WithDetail("disk offline"))

	if got := err.Detail(); got != "disk offline" {
		t.Fatalf("unexpected detail text: got %q", got)
	}
}

func TestWithReasonSetsReasonText(t *testing.T) {
	err := New(OperationFailed, "write failed", WithReason("quota-exceeded"))

	if got := err.Reason(); got != "quota-exceeded" {
		t.Fatalf("unexpected reason text: got %q", got)
	}
}

func TestErrorStringIncludesDetail(t *testing.T) {
	err := New(OperationFailed, "write failed", WithDetail("disk offline"))

	if got := err.Error(); !strings.Contains(got, "detail=disk offline") {
		t.Fatalf("expected detail in error string, got %q", got)
	}
}

func TestErrorJsonRoundTrip(t *testing.T) {
	err := New(OperationFailed, "write failed", WithReason("quota-exceeded"), WithDetail("disk offline"))

	payload := EncodeError(err, vcode.MustMarshalJson)
	got, decodeErr := DecodeError(payload, json.Unmarshal)
	if decodeErr != nil {
		t.Fatalf("DecodeError() error = %v", decodeErr)
	}
	if got.Code() != err.Code() || got.Message() != err.Message() || got.Reason() != err.Reason() || got.Detail() != err.Detail() {
		t.Fatalf("unexpected decoded error: got=%s/%s/%s/%s", got.Code(), got.Message(), got.Reason(), got.Detail())
	}
}

func TestClearErrorDetail(t *testing.T) {
	err := New(OperationFailed, "write failed", WithReason("quota-exceeded"), WithDetail("disk offline"))

	payload, decodeErr := ClearErrorDetail(EncodeError(err, vcode.MustMarshalJson), json.Unmarshal, vcode.MustMarshalJson)
	if decodeErr != nil {
		t.Fatalf("ClearErrorDetail() error = %v", decodeErr)
	}
	got, decodeErr := DecodeError(payload, json.Unmarshal)
	if decodeErr != nil {
		t.Fatalf("DecodeError() error = %v", decodeErr)
	}
	if got.Code() != err.Code() || got.Message() != err.Message() || got.Reason() != err.Reason() || got.Detail() != "" {
		t.Fatalf("unexpected decoded error: got=%s/%s/%s/%s", got.Code(), got.Message(), got.Reason(), got.Detail())
	}
}

func TestDecodeErrorRejectsUnknownCode(t *testing.T) {
	got, decodeErr := DecodeError([]byte(`{"code":"BAD","message":"boom","detail":"detail"}`), json.Unmarshal)
	if got != nil {
		t.Fatalf("expected nil decoded error, got %#v", got)
	}
	if decodeErr == nil || decodeErr.Error() != "unknown Code=BAD" {
		t.Fatalf("unexpected decode error: %v", decodeErr)
	}
}

func TestDecodeErrorReturnsDecodeError(t *testing.T) {
	got, decodeErr := DecodeError([]byte(`{`), json.Unmarshal)
	if got != nil {
		t.Fatalf("expected nil decoded error, got %#v", got)
	}
	if decodeErr == nil {
		t.Fatalf("unexpected decode error: %#v", decodeErr)
	}
}

func TestPanicNewFuncIfNotIsLazy(t *testing.T) {
	called := false

	PanicNewFuncIfNot(true, OperationFailed, func() string {
		called = true
		return "should not be called"
	})

	if called {
		t.Fatal("expected message function not to be called")
	}
}

func TestPanicNewFuncIfNotAppliesOptions(t *testing.T) {
	defer func() {
		recovered := recover()
		err, ok := recovered.(Error)
		if !ok {
			t.Fatalf("unexpected panic value: %#v", recovered)
		}
		if got := err.Code(); got != OperationFailed {
			t.Fatalf("unexpected code: got %s want %s", got, OperationFailed)
		}
		if got := err.Message(); got != "write failed" {
			t.Fatalf("unexpected message: %q", got)
		}
		if got := err.Detail(); got != "disk offline" {
			t.Fatalf("unexpected detail: %q", got)
		}
	}()

	PanicNewFuncIfNot(false, OperationFailed, func() string {
		return "write failed"
	}, WithDetail("disk offline"))
}

func TestPanicIfErrorCapturesSystemErrorStackWithoutMutatingOriginal(t *testing.T) {
	original := New(InvalidRequest, "bad request")
	recovered := recoverValue(func() { PanicIfError(original) })
	err, ok := recovered.(Error)
	if !ok {
		t.Fatalf("unexpected panic value: %#v", recovered)
	}
	if err == original {
		t.Fatal("expected raised error to be cloned")
	}
	if stack := PanicStack(original); stack != "" {
		t.Fatalf("original error unexpectedly has panic stack: %s", stack)
	}
	stack := PanicStack(err)
	if !strings.Contains(stack, "TestPanicIfErrorCapturesSystemErrorStackWithoutMutatingOriginal") {
		t.Fatalf("panic stack does not contain raise call site: %s", stack)
	}
	if strings.Contains(stack, "internal/core/ex.Panic") {
		t.Fatalf("panic stack contains helper frame: %s", stack)
	}
}

func TestPanicNewDoesNotCaptureApplicationErrorStack(t *testing.T) {
	recovered := recoverValue(func() { PanicNew(NotFound, "missing user") })
	err, ok := recovered.(Error)
	if !ok {
		t.Fatalf("unexpected panic value: %#v", recovered)
	}
	if stack := PanicStack(err); stack != "" {
		t.Fatalf("application error unexpectedly has panic stack: %s", stack)
	}
}

func TestRaisedSystemErrorDoesNotSerializePanicStack(t *testing.T) {
	recovered := recoverValue(func() { PanicNew(InvalidRequest, "bad request") })
	payload, err := json.Marshal(recovered)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "stack") || strings.Contains(string(payload), "error_test.go") {
		t.Fatalf("serialized error leaks panic stack: %s", payload)
	}
}

func TestRecoverApplicationReturnsApplicationError(t *testing.T) {
	want := New(NotFound, "missing user")
	got := RecoverApplication(want)

	if got != want {
		t.Fatalf("unexpected recovered error: got %#v want %#v", got, want)
	}
}

func TestRecoverReturnsSystemError(t *testing.T) {
	want := New(Internal, "boom")
	got := Recover(want)

	if got != want {
		t.Fatalf("unexpected recovered error: got %#v want %#v", got, want)
	}
}

func TestRecoverApplicationRepanicsSystemError(t *testing.T) {
	panicValue := New(Internal, "boom")

	defer func() {
		recovered := recover()
		if recovered != panicValue {
			t.Fatalf("unexpected panic value: got %#v want %#v", recovered, panicValue)
		}
	}()

	_ = RecoverApplication(panicValue)
}

func TestRecoverApplicationRepanicsNonExPanic(t *testing.T) {
	panicValue := "boom"

	defer func() {
		recovered := recover()
		if recovered != panicValue {
			t.Fatalf("unexpected panic value: got %#v want %#v", recovered, panicValue)
		}
	}()

	_ = RecoverApplication(panicValue)
}

func recoverValue(fn func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	fn()
	return nil
}
