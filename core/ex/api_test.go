package ex_test

import (
	"errors"
	"testing"

	"go.yorun.ai/vine/core/ex"
)

func TestFacadeCreatesStructuredError(t *testing.T) {
	cause := errors.New("database unavailable")
	err := ex.New(
		ex.OperationFailed,
		"save failed",
		ex.WithReason("storage_error"),
		ex.WithDetail("retry later"),
		ex.WithCause(cause),
	)

	if err.Code() != ex.OperationFailed || err.Type() != ex.ApplicationError {
		t.Fatalf("unexpected error classification: code=%s type=%s", err.Code(), err.Type())
	}
	if err.Message() != "save failed" || err.Reason() != "storage_error" || err.Detail() != "retry later" {
		t.Fatalf("unexpected structured error: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected structured error to retain its cause")
	}

	parsed, parseErr := ex.ParseCode(string(err.Code()))
	if parseErr != nil {
		t.Fatalf("ParseCode() error = %v", parseErr)
	}
	if parsed != err.Code() {
		t.Fatalf("ParseCode() = %s, want %s", parsed, err.Code())
	}
}
