package ex

import (
	"fmt"
)

type Type string

const (
	InvalidType      Type = "INVALID"
	NoError          Type = "OK"
	SystemError      Type = "SYSTEM"
	ApplicationError Type = "APPLICATION"
)

type Category string

const (
	InvalidCategory     Category = "INVALID"
	SuccessCategory     Category = "SUCCESS"
	FrameworkCategory   Category = "FRAMEWORK"
	InvocationCategory  Category = "INVOCATION"
	FallbackCategory    Category = "FALLBACK"
	ApplicationCategory Category = "APPLICATION"
)

type Code string

const (
	OK Code = "OK"

	// raised by rpc server/gateway
	ServiceUnavailable Code = "SERVICE_UNAVAILABLE"
	GatewayTimeout     Code = "GATEWAY_TIMEOUT"
	ClientForbidden    Code = "CLIENT_FORBIDDEN"
	InvalidRequest     Code = "INVALID_REQUEST"
	InvalidEvent       Code = "INVALID_EVENT"
	InvalidTask        Code = "INVALID_TASK"
	// raised by rpc client
	ServerUnreachable   Code = "SERVER_UNREACHABLE"
	InvocationCancelled Code = "INVOCATION_CANCELLED"
	InvocationTimeout   Code = "INVOCATION_TIMEOUT"
	InvocationFailed    Code = "INVOCATION_FAILED"
	UnexpectedResponse  Code = "UNEXPECTED_RESPONSE"
	// wrapped errors, should not raise directly
	Internal Code = "INTERNAL"
	Unknown  Code = "UNKNOWN"

	Unauthorized      Code = "UNAUTHORIZED"
	PermissionDenied  Code = "PERMISSION_DENIED"
	ElevationRequired Code = "ELEVATION_REQUIRED"
	ValidationFailed  Code = "VALIDATION_FAILED"
	OperationFailed   Code = "OPERATION_FAILED"
	NotFound          Code = "NOT_FOUND"
)

type _CodeMeta struct {
	kind           Type
	category       Category
	unresponsive   bool
	canRaiseDirect bool
	defaultMessage string
}

var codeMetas = map[Code]_CodeMeta{
	OK: {kind: NoError, category: SuccessCategory, canRaiseDirect: true},

	// raised by rpc server/gateway
	ServiceUnavailable: {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},
	GatewayTimeout:     {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},
	ClientForbidden:    {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},
	InvalidRequest:     {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},
	InvalidEvent:       {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},
	InvalidTask:        {kind: SystemError, category: FrameworkCategory, canRaiseDirect: true},

	// raised by rpc client
	ServerUnreachable:   {kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirect: true},
	InvocationCancelled: {kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirect: true},
	InvocationTimeout:   {kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirect: true},
	InvocationFailed:    {kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirect: true},
	UnexpectedResponse:  {kind: SystemError, category: InvocationCategory, unresponsive: true, canRaiseDirect: true},

	// wrapped errors, should not raise directly
	Internal: {kind: SystemError, category: FallbackCategory, canRaiseDirect: false, defaultMessage: "error occurred, please retry"},
	Unknown:  {kind: SystemError, category: FallbackCategory, canRaiseDirect: false, defaultMessage: "unknown error"},

	Unauthorized:      {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
	PermissionDenied:  {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
	ElevationRequired: {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
	ValidationFailed:  {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
	OperationFailed:   {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
	NotFound:          {kind: ApplicationError, category: ApplicationCategory, canRaiseDirect: true},
}

func (c Code) IsValid() bool {
	_, ok := codeMetas[c]
	return ok
}

func (c Code) IsUnresponsive() bool {
	meta, ok := codeMetas[c]
	return ok && meta.unresponsive
}

func (c Code) Category() Category {
	meta, ok := codeMetas[c]
	if !ok {
		return InvalidCategory
	}
	return meta.category
}

func (c Code) CanRaiseDirectly() bool {
	meta, ok := codeMetas[c]
	return ok && meta.canRaiseDirect
}

func (c Code) DefaultMessage() string {
	meta, ok := codeMetas[c]
	if !ok {
		return ""
	}
	return meta.defaultMessage
}

func ParseCode(codeStr string) (Code, error) {
	code := Code(codeStr)
	if code.IsValid() {
		return code, nil
	}
	return "", fmt.Errorf("invalid code=%s", codeStr)
}

func (c Code) Type() Type {
	meta, ok := codeMetas[c]
	if !ok {
		return InvalidType
	}
	return meta.kind
}
