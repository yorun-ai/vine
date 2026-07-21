package skel

import (
	"time"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	internalskel "go.yorun.ai/vine/internal/core/skel"
)

// Decimal is the runtime representation of the Skel decimal scalar.
type Decimal = internalskel.Decimal

// Binary is the runtime representation of the Skel binary scalar.
type Binary = internalskel.Binary

// PermissionCode is a stable permission identifier generated from a Skel contract.
type PermissionCode = internalskel.PermissionCode

// UUID is the runtime representation of the Skel uuid scalar.
type UUID = internalskel.UUID

// JSON is the runtime representation of arbitrary Skel JSON data.
type JSON = internalskel.JSON

// Timestamp is the runtime representation of an absolute Skel timestamp.
type Timestamp = internalskel.Timestamp

// Duration is the runtime representation of a Skel duration.
type Duration = internalskel.Duration

// LocalDate is the runtime representation of a date without a time zone.
type LocalDate = internalskel.LocalDate

// LocalTime is the runtime representation of a time of day without a time zone.
type LocalTime = internalskel.LocalTime

// LocalDateTime is the runtime representation of a local date and time without a time zone.
type LocalDateTime = internalskel.LocalDateTime

// Actor is the Skel wire representation of an actor.
type Actor = internalskel.Actor

// ActorBase contains fields common to generated actor values.
type ActorBase = internalskel.ActorBase

// ActorVia identifies the channel through which an actor entered the system.
type ActorVia = internalskel.ActorVia

// AuthMode describes whether a generated endpoint requires authentication.
type AuthMode = internalskel.AuthMode

// GeneratedInfo records the skelc version and source metadata of generated code.
type GeneratedInfo = internalskel.GeneratedInfo

// DomainSchema is the complete generated schema for one Skel domain.
type DomainSchema = internalskel.DomainSchema

// EnumSchema describes a generated Skel enum.
type EnumSchema = internalskel.EnumSchema

// EnumItemSchema describes one item in an EnumSchema.
type EnumItemSchema = internalskel.EnumItemSchema

// DataSchema describes a generated Skel data type.
type DataSchema = internalskel.DataSchema

// ConfigSchema describes a generated application configuration type.
type ConfigSchema = internalskel.ConfigSchema

// WebSchema describes a generated Web contract.
type WebSchema = internalskel.WebSchema

// EventSchema describes a generated event contract.
type EventSchema = internalskel.EventSchema

// ActorSchema describes a generated actor type.
type ActorSchema = internalskel.ActorSchema

// ActorAudienceSchema describes an audience accepted by an actor.
type ActorAudienceSchema = internalskel.ActorAudienceSchema

// ResourceSchema describes a permission-controlled resource.
type ResourceSchema = internalskel.ResourceSchema

// ResourceActionSchema describes an action supported by a resource.
type ResourceActionSchema = internalskel.ResourceActionSchema

// ResourceCheckSchema describes a generated resource authorization check.
type ResourceCheckSchema = internalskel.ResourceCheckSchema

// ServiceSchema describes a generated Rpc service.
type ServiceSchema = internalskel.ServiceSchema

// MethodSchema describes one method in a ServiceSchema.
type MethodSchema = internalskel.MethodSchema

// PermRequireMode controls how a generated permission requirement is evaluated.
type PermRequireMode = internalskel.PermRequireMode

// PermRequire describes a generated permission requirement.
type PermRequire = internalskel.PermRequire

// PermExpr is a generated permission expression.
type PermExpr = internalskel.PermExpr

// PermCheckInvocation describes a permission check against invocation metadata.
type PermCheckInvocation = internalskel.PermCheckInvocation

// PermCheckArgument describes a permission check against a method argument.
type PermCheckArgument = internalskel.PermCheckArgument

// TaskSchema describes a generated task contract.
type TaskSchema = internalskel.TaskSchema

// TriggerSchema describes a generated task trigger.
type TriggerSchema = internalskel.TriggerSchema

// MemberSchema describes one member of a generated structured type.
type MemberSchema = internalskel.MemberSchema

// TypeSchema describes a generated Skel type expression.
type TypeSchema = internalskel.TypeSchema

// TypeKind classifies a generated type expression.
type TypeKind = internalskel.TypeKind

// Scalar identifies a built-in Skel scalar type.
type Scalar = internalskel.Scalar

const (
	// ActorViaClient indicates that an actor came from a client request.
	ActorViaClient = internalskel.ActorViaClient
	// ActorViaAgent indicates that an actor came through an agent.
	ActorViaAgent = internalskel.ActorViaAgent
	// ActorViaOpenAPI indicates that an actor came through an OpenAPI entry.
	ActorViaOpenAPI = internalskel.ActorViaOpenAPI

	// AuthModeUnset leaves authentication behavior unspecified.
	AuthModeUnset = internalskel.AuthModeUnset
	// AuthModeAuth requires an authenticated actor.
	AuthModeAuth = internalskel.AuthModeAuth
	// AuthModeNoAuth permits unauthenticated access.
	AuthModeNoAuth = internalskel.AuthModeNoAuth

	// PermRequireModeCode requires a concrete permission code.
	PermRequireModeCode = internalskel.PermRequireModeCode
	// PermRequireModeCheck requires a generated permission check.
	PermRequireModeCheck = internalskel.PermRequireModeCheck
	// PermRequireModeAll requires every nested permission expression.
	PermRequireModeAll = internalskel.PermRequireModeAll
	// PermRequireModeAny requires at least one nested permission expression.
	PermRequireModeAny = internalskel.PermRequireModeAny

	// TypeKindScalar identifies a built-in scalar type.
	TypeKindScalar = internalskel.TypeKindScalar
	// TypeKindList identifies a list type.
	TypeKindList = internalskel.TypeKindList
	// TypeKindMap identifies a map type.
	TypeKindMap = internalskel.TypeKindMap
	// TypeKindEnum identifies a generated enum type.
	TypeKindEnum = internalskel.TypeKindEnum
	// TypeKindData identifies a generated structured data type.
	TypeKindData = internalskel.TypeKindData
	// TypeKindConfig identifies a generated configuration type.
	TypeKindConfig = internalskel.TypeKindConfig
	// TypeKindEvent identifies a generated event type.
	TypeKindEvent = internalskel.TypeKindEvent

	// TypeKindTypeParameter identifies a generic type parameter.
	TypeKindTypeParameter = internalskel.TypeKindTypeParameter
	// TypeKindSkelPermissionCode identifies the built-in permission-code type.
	TypeKindSkelPermissionCode = internalskel.TypeKindSkelPermissionCode

	// ScalarString identifies the string scalar.
	ScalarString = internalskel.ScalarString
	// ScalarBool identifies the boolean scalar.
	ScalarBool = internalskel.ScalarBool
	// ScalarInt identifies the machine-sized integer scalar.
	ScalarInt = internalskel.ScalarInt
	// ScalarLong identifies the 64-bit integer scalar.
	ScalarLong = internalskel.ScalarLong
	// ScalarFloat identifies the 32-bit floating-point scalar.
	ScalarFloat = internalskel.ScalarFloat
	// ScalarDouble identifies the 64-bit floating-point scalar.
	ScalarDouble = internalskel.ScalarDouble
	// ScalarDecimal identifies the arbitrary-precision decimal scalar.
	ScalarDecimal = internalskel.ScalarDecimal
	// ScalarJson identifies the arbitrary JSON scalar.
	ScalarJson = internalskel.ScalarJson
	// ScalarUuid identifies the UUID scalar.
	ScalarUuid = internalskel.ScalarUuid
	// ScalarTimestamp identifies the absolute timestamp scalar.
	ScalarTimestamp = internalskel.ScalarTimestamp
	// ScalarDuration identifies the duration scalar.
	ScalarDuration = internalskel.ScalarDuration
	// ScalarLocalDate identifies the local-date scalar.
	ScalarLocalDate = internalskel.ScalarLocalDate
	// ScalarLocalTime identifies the local-time scalar.
	ScalarLocalTime = internalskel.ScalarLocalTime
	// ScalarLocalDateTime identifies the local-date-time scalar.
	ScalarLocalDateTime = internalskel.ScalarLocalDateTime
	// ScalarBinary identifies the binary scalar.
	ScalarBinary = internalskel.ScalarBinary
)

// RegisterDomainSchema validates and adds schema to the process-wide domain registry.
func RegisterDomainSchema(schema *DomainSchema) {
	internalskel.RegisterDomainSchema(schema)
}

// RegisteredDomainSchemas returns registered domains in stable domain-name order.
func RegisteredDomainSchemas() []*DomainSchema {
	return internalskel.RegisteredDomainSchemas()
}

// NewDecimal converts value to the Skel decimal representation.
func NewDecimal(value decimal.Decimal) Decimal {
	return internalskel.NewDecimal(value)
}

// NewTimestamp converts t to the Skel timestamp representation.
func NewTimestamp(t time.Time) Timestamp {
	return internalskel.NewTimestamp(t)
}

// NewDuration converts d to the Skel duration representation.
func NewDuration(d time.Duration) Duration {
	return internalskel.NewDuration(d)
}

// NewUUID converts id to the Skel UUID representation.
func NewUUID(id uuid.UUID) UUID {
	return internalskel.NewUUID(id)
}

// NewLocalDate converts date to the Skel local-date representation.
func NewLocalDate(date civil.Date) LocalDate {
	return internalskel.NewLocalDate(date)
}

// NewLocalDateOf extracts the local date from t.
func NewLocalDateOf(t time.Time) LocalDate {
	return internalskel.NewLocalDateOf(t)
}

// NewLocalTime converts clock to the Skel local-time representation.
func NewLocalTime(clock civil.Time) LocalTime {
	return internalskel.NewLocalTime(clock)
}

// NewLocalTimeOf extracts the local time from t.
func NewLocalTimeOf(t time.Time) LocalTime {
	return internalskel.NewLocalTimeOf(t)
}

// NewLocalDateTime converts dateTime to the Skel local-date-time representation.
func NewLocalDateTime(dateTime civil.DateTime) LocalDateTime {
	return internalskel.NewLocalDateTime(dateTime)
}

// NewLocalDateTimeOf extracts the local date and time from t.
func NewLocalDateTimeOf(t time.Time) LocalDateTime {
	return internalskel.NewLocalDateTimeOf(t)
}
