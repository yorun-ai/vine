package ex

import internalex "go.yorun.ai/vine/internal/core/ex"

// Type classifies an error as absent, system, or application-defined.
type Type = internalex.Type

// Category groups related error codes.
type Category = internalex.Category

// Code is a stable machine-readable Vine error code.
type Code = internalex.Code

// Error is Vine's structured error value.
type Error = internalex.Error

// ErrorOption adds optional reason, detail, or cause data to an Error.
type ErrorOption = internalex.ErrorOption

const (
	// InvalidType is the zero or unrecognized error type.
	InvalidType = internalex.InvalidType
	// NoError identifies the absence of an error.
	NoError = internalex.NoError
	// SystemError identifies an error produced by infrastructure or the framework.
	SystemError = internalex.SystemError
	// ApplicationError identifies an error intentionally returned by application code.
	ApplicationError = internalex.ApplicationError
)

const (
	// InvalidCategory is the zero or unrecognized error category.
	InvalidCategory = internalex.InvalidCategory
	// SuccessCategory contains successful result codes.
	SuccessCategory = internalex.SuccessCategory
	// FrameworkCategory contains framework failure codes.
	FrameworkCategory = internalex.FrameworkCategory
	// InvocationCategory contains call and transport failure codes.
	InvocationCategory = internalex.InvocationCategory
	// FallbackCategory contains errors recovered without a more specific category.
	FallbackCategory = internalex.FallbackCategory
	// ApplicationCategory contains application-defined failure codes.
	ApplicationCategory = internalex.ApplicationCategory
)

const (
	// OK represents successful execution.
	OK = internalex.OK
	// ServiceUnavailable indicates that a service cannot currently handle the request.
	ServiceUnavailable = internalex.ServiceUnavailable
	// GatewayTimeout indicates that a gateway did not receive a timely response.
	GatewayTimeout = internalex.GatewayTimeout
	// ClientForbidden indicates that the calling application is forbidden.
	ClientForbidden = internalex.ClientForbidden
	// InvalidRequest indicates malformed or semantically invalid input.
	InvalidRequest = internalex.InvalidRequest
	// ServerUnreachable indicates that the target server could not be reached.
	ServerUnreachable = internalex.ServerUnreachable
	// InvocationCancelled indicates cancellation by a context or caller.
	InvocationCancelled = internalex.InvocationCancelled
	// InvocationTimeout indicates that an invocation exceeded its deadline.
	InvocationTimeout = internalex.InvocationTimeout
	// InvocationFailed indicates an invocation failure without a more specific code.
	InvocationFailed = internalex.InvocationFailed
	// UnexpectedResponse indicates a response that does not match the contract.
	UnexpectedResponse = internalex.UnexpectedResponse
	// Internal indicates an unexpected internal failure.
	Internal = internalex.Internal
	// Unknown indicates that no more specific error code is available.
	Unknown = internalex.Unknown
	// Unauthorized indicates that authentication is required or invalid.
	Unauthorized = internalex.Unauthorized
	// PermissionDenied indicates that the actor lacks a required permission.
	PermissionDenied = internalex.PermissionDenied
	// ElevationRequired indicates that the operation requires elevated authority.
	ElevationRequired = internalex.ElevationRequired
	// ValidationFailed indicates that application validation rejected input.
	ValidationFailed = internalex.ValidationFailed
	// OperationFailed indicates that an application operation could not complete.
	OperationFailed = internalex.OperationFailed
	// NotFound indicates that a requested resource does not exist.
	NotFound = internalex.NotFound
)

// ParseCode parses the canonical string representation of an error code.
func ParseCode(codeStr string) (Code, error) {
	return internalex.ParseCode(codeStr)
}

// WithReason attaches a stable machine-oriented reason to an Error.
func WithReason(reason string) ErrorOption {
	return internalex.WithReason(reason)
}

// WithDetail attaches diagnostic detail to an Error.
func WithDetail(detail string) ErrorOption {
	return internalex.WithDetail(detail)
}

// WithCause attaches causeError as the underlying cause of an Error.
func WithCause(causeError error) ErrorOption {
	return internalex.WithCause(causeError)
}

// New creates a structured Error with code and message.
func New(code Code, message string, opts ...ErrorOption) Error {
	return internalex.New(code, message, opts...)
}

// F formats a message using the package's error formatting convention.
func F(template string, args ...any) string {
	return internalex.F(template, args...)
}

// NewOK creates an Error representing successful execution.
func NewOK() Error {
	return internalex.NewOK()
}

// NewInternal creates a generic internal framework Error.
func NewInternal() Error {
	return internalex.NewInternal()
}

// DecodeError decodes a structured Error from payload using unmarshal.
func DecodeError(payload []byte, unmarshal func([]byte, any) error) (Error, error) {
	return internalex.DecodeError(payload, unmarshal)
}

// EncodeError encodes err using mustMarshal.
func EncodeError(err Error, mustMarshal func(any) []byte) []byte {
	return internalex.EncodeError(err, mustMarshal)
}

// ClearErrorDetail removes diagnostic detail from an encoded Error payload.
func ClearErrorDetail(payload []byte, unmarshal func([]byte, any) error, mustMarshal func(any) []byte) ([]byte, error) {
	return internalex.ClearErrorDetail(payload, unmarshal, mustMarshal)
}

// PanicIfError panics with err when err is non-nil.
func PanicIfError(err error) {
	internalex.PanicIfError(err)
}

// PanicNew panics with a newly constructed structured Error.
func PanicNew(code Code, message string, opts ...ErrorOption) {
	internalex.PanicNew(code, message, opts...)
}

// PanicNewIfError panics with code and err as the cause when err is non-nil.
func PanicNewIfError(err error, code Code) {
	internalex.PanicNewIfError(err, code)
}

// PanicNewIfNot panics with a new Error when condition is false.
func PanicNewIfNot(condition bool, code Code, message string, opts ...ErrorOption) {
	internalex.PanicNewIfNot(condition, code, message, opts...)
}

// PanicNewFuncIfNot lazily constructs and panics with an Error when condition is false.
func PanicNewFuncIfNot(condition bool, code Code, messageFunc func() string, opts ...ErrorOption) {
	internalex.PanicNewFuncIfNot(condition, code, messageFunc, opts...)
}

// Recover converts a recovered panic value into a structured Error.
func Recover(r any) Error {
	return internalex.Recover(r)
}

// RecoverApplication converts a recovered application panic into a structured Error.
func RecoverApplication(r any) Error {
	return internalex.RecoverApplication(r)
}
