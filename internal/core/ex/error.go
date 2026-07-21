package ex

import (
	"fmt"
	"strings"

	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vstring"
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
	CodeValue    Code   `json:"code"`
	MessageValue string `json:"message"`

	ReasonValue string `json:"reason"`
	DetailValue string `json:"detail"`

	cause error
}

func (e *_Error) Error() string {
	parts := []string{
		fmt.Sprintf("type=%s", e.Type()),
		fmt.Sprintf("code=%s", e.CodeValue),
	}
	if e.MessageValue != "" {
		parts = append([]string{e.MessageValue}, parts...)
	}
	if e.DetailValue != "" {
		parts = append(parts, fmt.Sprintf("detail=%s", e.DetailValue))
	}
	return strings.Join(parts, " ")
}

func (e *_Error) Type() Type {
	return e.CodeValue.Type()
}

func (e *_Error) Code() Code {
	return e.CodeValue
}

func (e *_Error) Message() string {
	return e.MessageValue
}

func (e *_Error) Reason() string {
	return e.ReasonValue
}

func (e *_Error) Detail() string {
	return e.DetailValue
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
		e.ReasonValue = reason
	}
}

func WithDetail(detail string) ErrorOption {
	vpre.CheckNot(vstring.IsBlank(detail), "missing detail value")
	return func(e *_Error) {
		e.DetailValue = detail
	}
}

func WithCause(cause error) ErrorOption {
	vpre.CheckNotNil(cause, "missing cause error")
	return func(err *_Error) {
		err.cause = cause
	}
}

// NewError

func F(template string, args ...any) string {
	return fmt.Sprintf(template, args...)
}

func New(code Code, message string, opts ...ErrorOption) Error {
	vpre.Check(code.IsValid(), "unknown Code=%s", code)
	err := &_Error{
		CodeValue:    code,
		MessageValue: message,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

func NewOK() Error {
	return New(OK, "")
}

func NewInternal() Error {
	return New(Internal, Internal.DefaultMessage())
}

// Serialization

func DecodeError(payload []byte, unmarshal func([]byte, any) error) (Error, error) {
	exErr, decodeErr := decodeError(payload, unmarshal)
	if decodeErr != nil {
		return nil, decodeErr
	}
	return exErr, nil
}

func EncodeError(err Error, mustMarshal func(any) []byte) []byte {
	return mustMarshal(err)
}

func ClearErrorDetail(payload []byte, unmarshal func([]byte, any) error, mustMarshal func(any) []byte) ([]byte, error) {
	exErr, decodeErr := decodeError(payload, unmarshal)
	if decodeErr != nil {
		return nil, decodeErr
	}
	exErr.DetailValue = ""
	return mustMarshal(exErr), nil
}

func decodeError(payload []byte, unmarshal func([]byte, any) error) (*_Error, error) {
	var exErr *_Error
	if err := unmarshal(payload, &exErr); err != nil {
		return nil, err
	}
	vpre.Check(exErr != nil, "invalid error payload")

	if !exErr.CodeValue.IsValid() {
		return nil, fmt.Errorf("unknown Code=%s", exErr.CodeValue)
	}

	return exErr, nil
}

// Panic

// Keep these conditional helpers panicking in place so the captured stack
// points at the original call site instead of an extra wrapper frame.

func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func PanicNew(code Code, message string, opts ...ErrorOption) {
	panic(New(code, message, opts...))
}

func PanicNewIfError(err error, code Code) {
	if err != nil {
		panic(New(code, err.Error()))
	}
}

func PanicNewIfNot(condition bool, code Code, message string, opts ...ErrorOption) {
	if !condition {
		panic(New(code, message, opts...))
	}
}

func PanicNewFuncIfNot(condition bool, code Code, messageFunc func() string, opts ...ErrorOption) {
	if !condition {
		panic(New(code, messageFunc(), opts...))
	}
}

// Recover

func Recover(r any) Error {
	return recoverError(r, true)
}

func RecoverApplication(r any) Error {
	return recoverError(r, false)
}

func recoverError(r any, includeSysErr bool) Error {
	if r == nil {
		return nil
	}
	if err, ok := r.(Error); ok {
		if includeSysErr || err.Type() == ApplicationError {
			return err
		}
	}
	panic(r)
}
