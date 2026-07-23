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

func TestPanicIfErrorClonesWithoutReplacingExistingStack(t *testing.T) {
	original := New(InvalidRequest, "bad request")
	originalStack := Stack(original)
	recovered := recoverValue(func() { PanicIfError(original) })
	err, ok := recovered.(Error)
	if !ok {
		t.Fatalf("unexpected panic value: %#v", recovered)
	}
	if err == original {
		t.Fatal("expected raised error to be cloned")
	}
	if Stack(err) != originalStack {
		t.Fatalf("raise replaced the existing error stack\nraised=%s\noriginal=%s", Stack(err), originalStack)
	}
	if _, panicked := PanicValue(original); panicked {
		t.Fatal("original error was mutated with panic diagnostics")
	}
	if panicValue, panicked := PanicValue(err); !panicked || !strings.Contains(panicValue, "bad request") {
		t.Fatalf("unexpected panic diagnostic: value=%q panicked=%t", panicValue, panicked)
	}
}

func TestPanicNewKeepsApplicationErrorStackAndPanicValue(t *testing.T) {
	recovered := recoverValue(func() { PanicNew(NotFound, "missing user") })
	err, ok := recovered.(Error)
	if !ok {
		t.Fatalf("unexpected panic value: %#v", recovered)
	}
	if stack := Stack(err); !strings.Contains(stack, "TestPanicNewKeepsApplicationErrorStackAndPanicValue") {
		t.Fatalf("unexpected application error stack: %s", stack)
	}
	if panicValue, panicked := PanicValue(err); !panicked || !strings.Contains(panicValue, "missing user") {
		t.Fatalf("unexpected panic diagnostic: value=%q panicked=%t", panicValue, panicked)
	}
}

func TestRaisedErrorDoesNotSerializeLocalDiagnostics(t *testing.T) {
	recovered := recoverValue(func() { PanicNew(InvalidRequest, "bad request") })
	payload, err := json.Marshal(recovered)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "stack") || strings.Contains(string(payload), "panic") || strings.Contains(string(payload), "error_test.go") {
		t.Fatalf("serialized error leaks local diagnostics: %s", payload)
	}
}

func TestRecoverApplicationReturnsApplicationError(t *testing.T) {
	want := New(NotFound, "missing user")
	got := RecoverApplication(want)

	if got == want || got.Code() != want.Code() || got.Message() != want.Message() {
		t.Fatalf("unexpected recovered error: got %#v want equivalent clone of %#v", got, want)
	}
	if _, panicked := PanicValue(got); !panicked {
		t.Fatal("recovered application error is missing panic diagnostics")
	}
}

func TestRecoverReturnsSystemError(t *testing.T) {
	want := New(Internal, "boom")
	got := Recover(want)

	if got == want || got.Code() != want.Code() || got.Message() != want.Message() {
		t.Fatalf("unexpected recovered error: got %#v want equivalent clone of %#v", got, want)
	}
	if _, panicked := PanicValue(got); !panicked {
		t.Fatal("recovered system error is missing panic diagnostics")
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

func TestNewCapturesLocalStackWithoutSerializingIt(t *testing.T) {
	err := New(OperationFailed, "boom")
	stack := Stack(err)
	if !strings.Contains(stack, "TestNewCapturesLocalStackWithoutSerializingIt") {
		t.Fatalf("unexpected stack: %s", stack)
	}

	payload, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("json.Marshal() error = %v", marshalErr)
	}
	if strings.Contains(string(payload), "stack") || strings.Contains(string(payload), "TestNewCaptures") {
		t.Fatalf("local stack leaked into wire payload: %s", payload)
	}

	decoded, decodeErr := DecodeError(payload, json.Unmarshal)
	if decodeErr != nil {
		t.Fatalf("DecodeError() error = %v", decodeErr)
	}
	if Stack(decoded) != "" {
		t.Fatalf("remote decoded error should not have a local stack: %s", Stack(decoded))
	}
}

func TestWithCausePreservesCauseStack(t *testing.T) {
	cause := New(OperationFailed, "source")
	wrapper := New(Internal, "wrapper", WithCause(cause))
	if Stack(wrapper) != Stack(cause) {
		t.Fatalf("wrapper should preserve the source stack\nwrapper=%s\ncause=%s", Stack(wrapper), Stack(cause))
	}
}

func TestPanickedErrorKeepsBothStacksAndPrefersErrorStack(t *testing.T) {
	source := New(OperationFailed, "source")
	wantStack := Stack(source)

	recovered := recoverValue(func() { panic(source) })
	err := RecoverExecution(recovered)
	internalErr := err.(*_Error)
	if len(internalErr.errorStack) == 0 {
		t.Fatal("panicked Error lost its error stack")
	}
	if len(internalErr.panicStack) == 0 {
		t.Fatal("panicked Error is missing its panic stack")
	}
	if got := Stack(err); got != wantStack {
		t.Fatalf("stack should prefer the Error source\ngot=%s\nwant=%s", got, wantStack)
	}
}

func TestRecoverExecutionCapturesRawPanicStackAndSafeValue(t *testing.T) {
	var recovered Error
	func() {
		defer func() {
			recovered = RecoverExecution(recover())
		}()
		panic(struct{ Secret string }{Secret: "hidden"})
	}()

	if recovered.Code() != Internal {
		t.Fatalf("unexpected recovered code: %s", recovered.Code())
	}
	panicValue, panicked := PanicValue(recovered)
	if !panicked || panicValue != "<struct { Secret string }>" {
		t.Fatalf("unexpected panic value: value=%q panicked=%t", panicValue, panicked)
	}
	if stack := Stack(recovered); !strings.Contains(stack, "TestRecoverExecutionCapturesRawPanicStackAndSafeValue") {
		t.Fatalf("unexpected panic stack: %s", stack)
	}
	internalErr := recovered.(*_Error)
	if len(internalErr.errorStack) != 0 || len(internalErr.panicStack) == 0 {
		t.Fatalf("raw panic should use only panic stack: %#v", internalErr)
	}
}

func TestRecoverExecutionNormalizesPanickedOKError(t *testing.T) {
	recovered := RecoverExecution(NewOK())
	if recovered.Code() != Internal {
		t.Fatalf("unexpected recovered code: %s", recovered.Code())
	}
	if _, panicked := PanicValue(recovered); !panicked {
		t.Fatal("normalized panic is missing diagnostics")
	}
}

func recoverValue(fn func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	fn()
	return nil
}
