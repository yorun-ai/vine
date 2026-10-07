package ex

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vstring"
	rpchttp "go.yorun.ai/vrpc/transport/http"
)

// Error

type Error interface {
	error

	Type() Type
	Code() Code
	Message() string

	Reason() string
	Detail() string

	mustBeError()
}

type _Error struct {
	code    Code
	message string

	reason string
	detail string

	cause      error
	errorStack []uintptr
	panicStack []uintptr
	panicked   bool
	panicValue string
}

func (e *_Error) Error() string {
	parts := []string{
		fmt.Sprintf("type=%s", e.Type()),
		fmt.Sprintf("code=%s", e.code),
	}
	if e.message != "" {
		parts = append([]string{e.message}, parts...)
	}
	if e.detail != "" {
		parts = append(parts, fmt.Sprintf("detail=%s", e.detail))
	}
	return strings.Join(parts, " ")
}

func (e *_Error) Type() Type {
	return e.code.Type()
}

func (e *_Error) Code() Code {
	return e.code
}

func (e *_Error) Message() string {
	return e.message
}

func (e *_Error) Reason() string {
	return e.reason
}

func (e *_Error) Detail() string {
	return e.detail
}

func (e *_Error) Unwrap() error {
	return e.cause
}

func (e *_Error) mustBeError() {}

// ErrorOption

type ErrorOption func(*_Error)

func WithReason(reason string) ErrorOption {
	vpre.CheckNot(vstring.IsBlank(reason), "missing reason value")
	return func(e *_Error) {
		e.reason = reason
	}
}

func WithDetail(detail string) ErrorOption {
	vpre.CheckNot(vstring.IsBlank(detail), "missing detail value")
	return func(e *_Error) {
		e.detail = detail
	}
}

func WithCause(cause error) ErrorOption {
	vpre.CheckNotNil(cause, "missing cause error")
	return func(err *_Error) {
		err.cause = cause
		if causeErr, ok := cause.(*_Error); ok {
			if len(causeErr.errorStack) > 0 {
				err.errorStack = append([]uintptr(nil), causeErr.errorStack...)
			} else if len(causeErr.panicStack) > 0 {
				err.errorStack = nil
			}
			err.panicStack = append([]uintptr(nil), causeErr.panicStack...)
			err.panicked = causeErr.panicked
			err.panicValue = causeErr.panicValue
		}
	}
}

// NewError

func F(template string, args ...any) string {
	return fmt.Sprintf(template, args...)
}

func New(code Code, message string, opts ...ErrorOption) Error {
	vpre.Check(code.IsValid(), "unknown Code=%s", code)
	err := &_Error{
		code:    code,
		message: message,
	}
	if code != OK {
		err.errorStack = captureStack()
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

var okError = new(_Error{
	code: OK,
})

func NewOK() Error {
	return okError
}

func NewInternal() Error {
	return New(Internal, Internal.DefaultMessage())
}

// Serialization

// ToPayload copies the transport fields without exposing local diagnostics.
func ToPayload(err Error) *rpchttp.ErrorPayload {
	if err == nil {
		return nil
	}
	return new(rpchttp.ErrorPayload{
		Code:    string(err.Code()),
		Message: err.Message(),
		Reason:  err.Reason(),
		Detail:  err.Detail(),
	})
}

// FromPayload validates the Vine error code without creating a local stack.
func FromPayload(payload *rpchttp.ErrorPayload) (Error, error) {
	if payload == nil {
		return nil, fmt.Errorf("invalid error payload")
	}
	code, err := ParseCode(payload.Code)
	if err != nil {
		return nil, err
	}
	return new(_Error{
		code:    code,
		message: payload.Message,
		reason:  payload.Reason,
		detail:  payload.Detail,
	}), nil
}

// MarshalJSON retains direct serialization using the shared transport fields.
func (e *_Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(ToPayload(e))
}

// MarshalCBOR retains direct serialization using the shared transport fields.
func (e *_Error) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(ToPayload(e))
}

// Panic

// Keep conditional checks in these helpers so the panic stack can be captured
// from the original call site without another conditional wrapper frame.

func PanicIfError(err error) {
	if err != nil {
		panicWithStack(err)
	}
}

func PanicNew(code Code, message string, opts ...ErrorOption) {
	panicWithStack(New(code, message, opts...))
}

func PanicNewIfError(err error, code Code) {
	if err != nil {
		panicWithStack(New(code, err.Error()))
	}
}

func PanicNewIfNot(condition bool, code Code, message string, opts ...ErrorOption) {
	if !condition {
		panicWithStack(New(code, message, opts...))
	}
}

func PanicNewFuncIfNot(condition bool, code Code, messageFunc func() string, opts ...ErrorOption) {
	if !condition {
		panicWithStack(New(code, messageFunc(), opts...))
	}
}

func panicWithStack(err error) {
	exErr, ok := err.(*_Error)
	if !ok {
		panic(err)
	}
	panic(withPanic(exErr, err))
}

// Recover

func Recover(r any) Error {
	return recoverError(r, true)
}

func RecoverApplication(r any) Error {
	return recoverError(r, false)
}

// RecoverExecution converts any panic value recovered at an Rpc, Task, or
// Event execution boundary into a structured Error with local diagnostics.
func RecoverExecution(r any) Error {
	if r == nil {
		return nil
	}
	if err, ok := r.(Error); ok && err.Code() != OK {
		return withPanic(err, r)
	}
	return withPanic(newInternalWithoutStack(), r)
}

func recoverError(r any, includeSysErr bool) Error {
	if r == nil {
		return nil
	}
	if err, ok := r.(Error); ok {
		if includeSysErr || err.Type() == ApplicationError {
			if err.Code() == OK {
				return withPanic(newInternalWithoutStack(), r)
			}
			return withPanic(err, r)
		}
	}
	panic(r)
}

func withPanic(err Error, value any) Error {
	internalErr := cloneError(err)
	if len(internalErr.panicStack) == 0 {
		internalErr.panicStack = captureStack()
	}
	internalErr.panicked = true
	internalErr.panicValue = safePanicValue(value)
	return internalErr
}

func cloneError(err Error) *_Error {
	internalErr := err.(*_Error)
	return new(_Error{
		code:       internalErr.code,
		message:    internalErr.message,
		reason:     internalErr.reason,
		detail:     internalErr.detail,
		cause:      internalErr.cause,
		errorStack: append([]uintptr(nil), internalErr.errorStack...),
		panicStack: append([]uintptr(nil), internalErr.panicStack...),
		panicked:   internalErr.panicked,
		panicValue: internalErr.panicValue,
	})
}

func newInternalWithoutStack() *_Error {
	return new(_Error{
		code:    Internal,
		message: Internal.DefaultMessage(),
	})
}

// PanicValue returns the safe local representation of a recovered panic.
func PanicValue(err Error) (string, bool) {
	if err == nil {
		return "", false
	}
	internalErr, ok := err.(*_Error)
	if !ok || !internalErr.panicked {
		return "", false
	}
	return internalErr.panicValue, true
}

func Stack(err Error) string {
	if err == nil {
		return ""
	}
	internalErr, ok := err.(*_Error)
	if !ok {
		return ""
	}
	if len(internalErr.errorStack) > 0 {
		return formatStack(internalErr.errorStack)
	}
	return formatStack(internalErr.panicStack)
}

func captureStack() []uintptr {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	return pcs[:n]
}

func formatStack(pcs []uintptr) string {
	if len(pcs) == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs)
	var builder strings.Builder
	for {
		frame, more := frames.Next()
		if !isErrorFrame(frame) {
			_, _ = fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			break
		}
	}
	return strings.TrimSpace(builder.String())
}

func isErrorFrame(frame runtime.Frame) bool {
	if strings.HasSuffix(frame.File, "_test.go") {
		return false
	}
	return strings.HasPrefix(frame.Function, "go.yorun.ai/vine/internal/core/ex.") ||
		strings.HasPrefix(frame.Function, "go.yorun.ai/vine/core/ex.") ||
		strings.HasPrefix(frame.Function, "runtime.")
}

func safePanicValue(value any) (result string) {
	defer func() {
		if recover() != nil {
			result = "<panic value formatting failed>"
		}
	}()

	switch casted := value.(type) {
	case nil:
		result = "<nil>"
	case string:
		result = casted
	case bool:
		result = fmt.Sprintf("%t", casted)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64, complex64, complex128:
		result = fmt.Sprintf("%v", casted)
	case error:
		result = casted.Error()
	default:
		result = "<" + reflect.TypeOf(value).String() + ">"
	}
	const maxPanicValueBytes = 2 * 1024
	if len(result) > maxPanicValueBytes {
		result = result[:maxPanicValueBytes] + "<truncated>"
	}
	return result
}
