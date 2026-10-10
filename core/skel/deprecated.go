package skel

import (
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	"github.com/shopspring/decimal"
	"go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/core/skel/legacy"
	"go.yorun.ai/vine/util/vpre"
)

// Actor is the marker interface implemented by legacy generated actor types.
//
// Deprecated: Regenerate contracts with current skelc, which no longer emits actor marker types.
type Actor interface {
	Name() string
	SkelName() string
	Vias() []descriptor.ActorViaKind

	mustBeActor()
}

// ActorBase supplies the marker methods for legacy generated actor types.
//
// Deprecated: Regenerate contracts with current skelc, which no longer embeds ActorBase.
type ActorBase struct{}

func (ActorBase) Name() string {
	return ""
}

func (ActorBase) SkelName() string {
	return ""
}

func (ActorBase) Vias() []descriptor.ActorViaKind {
	return nil
}

func (ActorBase) mustBeActor() {}

// Sensitive is implemented by generated values that are sensitive as a whole.
//
// Deprecated: Use go.yorun.ai/skel/types.Sensitive instead.
type Sensitive = skeltype.Sensitive

// Decimal is the runtime representation of the Skel decimal scalar.
//
// Deprecated: Use go.yorun.ai/skel/types.Decimal instead.
type Decimal = skeltype.Decimal

// Binary is the runtime representation of the Skel binary scalar.
//
// Deprecated: Use go.yorun.ai/skel/types.Binary instead.
type Binary = skeltype.Binary

// UUID is the runtime representation of the Skel uuid scalar.
//
// Deprecated: Use go.yorun.ai/skel/types.UUID instead.
type UUID = skeltype.UUID

// JSON is the runtime representation of arbitrary Skel JSON data.
//
// Deprecated: Use go.yorun.ai/skel/types.JSON instead.
type JSON = skeltype.JSON

// Timestamp is the runtime representation of an absolute Skel timestamp.
//
// Deprecated: Use go.yorun.ai/skel/types.Timestamp instead.
type Timestamp = skeltype.Timestamp

// Duration is the runtime representation of a Skel duration.
//
// Deprecated: Use go.yorun.ai/skel/types.Duration instead.
type Duration = skeltype.Duration

// LocalDate is the runtime representation of a date without a time zone.
//
// Deprecated: Use go.yorun.ai/skel/types.LocalDate instead.
type LocalDate = skeltype.LocalDate

// LocalTime is the runtime representation of a time of day without a time zone.
//
// Deprecated: Use go.yorun.ai/skel/types.LocalTime instead.
type LocalTime = skeltype.LocalTime

// LocalDateTime is the runtime representation of a local date and time without a time zone.
//
// Deprecated: Use go.yorun.ai/skel/types.LocalDateTime instead.
type LocalDateTime = skeltype.LocalDateTime

// ActorVia identifies the channel through which an actor entered the system.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.ActorViaKind in new contracts.
type ActorVia = descriptor.ActorViaKind

// NewDecimal converts value to the Skel decimal representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewDecimal instead.
func NewDecimal(value decimal.Decimal) Decimal {
	return skeltype.NewDecimal(value)
}

// NewTimestamp converts t to the Skel timestamp representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewTimestamp instead.
func NewTimestamp(t time.Time) Timestamp {
	return skeltype.NewTimestamp(t)
}

// NewDuration converts d to the Skel duration representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewDuration instead.
func NewDuration(d time.Duration) Duration {
	return skeltype.NewDuration(d)
}

// NewUUID converts id to the Skel UUID representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewUUID instead.
func NewUUID(id uuid.UUID) UUID {
	return skeltype.NewUUID(id)
}

// NewLocalDate converts date to the Skel local-date representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalDate instead.
func NewLocalDate(date civil.Date) LocalDate {
	return skeltype.NewLocalDate(date)
}

// NewLocalDateOf extracts the local date from t.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalDateOf instead.
func NewLocalDateOf(t time.Time) LocalDate {
	return skeltype.NewLocalDateOf(t)
}

// NewLocalTime converts clock to the Skel local-time representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalTime instead.
func NewLocalTime(clock civil.Time) LocalTime {
	return skeltype.NewLocalTime(clock)
}

// NewLocalTimeOf extracts the local time from t.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalTimeOf instead.
func NewLocalTimeOf(t time.Time) LocalTime {
	return skeltype.NewLocalTimeOf(t)
}

// NewLocalDateTime converts dateTime to the Skel local-date-time representation.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalDateTime instead.
func NewLocalDateTime(dateTime civil.DateTime) LocalDateTime {
	return skeltype.NewLocalDateTime(dateTime)
}

// NewLocalDateTimeOf extracts the local date and time from t.
//
// Deprecated: Use go.yorun.ai/skel/types.NewLocalDateTimeOf instead.
func NewLocalDateTimeOf(t time.Time) LocalDateTime {
	return skeltype.NewLocalDateTimeOf(t)
}

const (
	// ActorViaClient identifies the client channel.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ActorViaClient instead.
	ActorViaClient = descriptor.ActorViaClient

	// ActorViaAgent identifies the agent channel.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ActorViaAgent instead.
	ActorViaAgent = descriptor.ActorViaAgent

	// ActorViaOpenAPI identifies the OpenAPI channel.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ActorViaOpenAPI instead.
	ActorViaOpenAPI = descriptor.ActorViaOpenAPI
)

// AuthMode controls Portal authentication for a generated endpoint.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.AuthMode in new contracts.
type AuthMode = legacy.AuthMode

// GeneratedInfo records the skelc version and source metadata of generated code.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.GeneratedInfo in new contracts.
type GeneratedInfo = legacy.GeneratedInfo

// DomainSchema accepts contracts from older generated code.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Domain in new contracts.
type DomainSchema = legacy.DomainSchema

// EnumSchema describes a generated Skel enum.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Enum in new contracts.
type EnumSchema = legacy.EnumSchema

// EnumItemSchema describes one item in an EnumSchema.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.EnumItem in new contracts.
type EnumItemSchema = legacy.EnumItemSchema

// DataSchema describes a generated Skel data type.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Data in new contracts.
type DataSchema = legacy.DataSchema

// ConfigSchema describes a generated application configuration type.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Config in new contracts.
type ConfigSchema = legacy.ConfigSchema

// WebSchema describes a generated Web contract.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Web in new contracts.
type WebSchema = legacy.WebSchema

// EventSchema describes a generated event contract.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Event in new contracts.
type EventSchema = legacy.EventSchema

// ActorSchema describes a generated actor type.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Actor in new contracts.
type ActorSchema = legacy.ActorSchema

// ActorAudienceSchema describes an audience accepted by an actor.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.ActorAudience in new contracts.
type ActorAudienceSchema = legacy.ActorAudienceSchema

// ResourceSchema describes a permission-controlled resource.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Resource in new contracts.
type ResourceSchema = legacy.ResourceSchema

// ResourceActionSchema describes an action supported by a resource.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.ResourceAction in new contracts.
type ResourceActionSchema = legacy.ResourceActionSchema

// ResourceCheckSchema describes a generated resource authorization check.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.ResourceCheck in new contracts.
type ResourceCheckSchema = legacy.ResourceCheckSchema

// ServiceSchema describes a generated Rpc service.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Service in new contracts.
type ServiceSchema = legacy.ServiceSchema

// MethodSchema describes one method in a ServiceSchema.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Method in new contracts.
type MethodSchema = legacy.MethodSchema

// PermRequireMode controls how a generated permission requirement is evaluated.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequireMode in new contracts.
type PermRequireMode = legacy.PermRequireMode

// PermRequire describes a generated permission requirement.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequire in new contracts.
type PermRequire = legacy.PermRequire

// PermExpr is a generated permission expression.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionExpression in new contracts.
type PermExpr = legacy.PermExpr

// PermCheckInvocation describes a permission check against invocation metadata.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionCheckInvocation in new contracts.
type PermCheckInvocation = legacy.PermCheckInvocation

// PermCheckArgument describes a permission check against a method argument.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionCheckArgument in new contracts.
type PermCheckArgument = legacy.PermCheckArgument

// TaskSchema describes a generated task contract.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Task in new contracts.
type TaskSchema = legacy.TaskSchema

// TriggerSchema describes a generated task trigger.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.TaskTrigger in new contracts.
type TriggerSchema = legacy.TriggerSchema

// MemberSchema describes one member of a generated structured type.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Member in new contracts.
type MemberSchema = legacy.MemberSchema

// TypeSchema describes a generated Skel type expression.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Type in new contracts.
type TypeSchema = legacy.TypeSchema

// TypeKind classifies a generated type expression.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKind in new contracts.
type TypeKind = legacy.TypeKind

// Scalar identifies a built-in Skel scalar type.
//
// Deprecated: Use go.yorun.ai/skel/descriptor.Scalar in new contracts.
type Scalar = legacy.Scalar

const (
	// AuthModeInherit uses the enclosing service authentication mode and is valid only on methods.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.AuthModeInherit instead.
	AuthModeInherit = descriptor.AuthModeInherit

	// AuthModeRequired requires valid credentials.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.AuthModeRequired instead.
	AuthModeRequired = descriptor.AuthModeRequired

	// AuthModeOptional permits missing credentials but rejects invalid credentials.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.AuthModeOptional instead.
	AuthModeOptional = descriptor.AuthModeOptional

	// AuthModeAnonymous permits only requests without credentials.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.AuthModeAnonymous instead.
	AuthModeAnonymous = descriptor.AuthModeAnonymous

	// AuthModeOff skips portal authentication for web, preserving native credentials.
	// Rpc services and methods cannot use this mode.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.AuthModeOff instead.
	AuthModeOff = descriptor.AuthModeOff

	// PermRequireModeCode requires a concrete permission code.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequireModeCode instead.
	PermRequireModeCode = legacy.PermRequireModeCode

	// PermRequireModeCheck requires a generated permission check.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequireModeCheck instead.
	PermRequireModeCheck = legacy.PermRequireModeCheck

	// PermRequireModeAll requires every nested permission expression.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequireModeAll instead.
	PermRequireModeAll = legacy.PermRequireModeAll

	// PermRequireModeAny requires at least one nested permission expression.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.PermissionRequireModeAny instead.
	PermRequireModeAny = legacy.PermRequireModeAny

	// TypeKindScalar identifies a built-in scalar type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindScalar instead.
	TypeKindScalar = legacy.TypeKindScalar

	// TypeKindList identifies a list type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindList instead.
	TypeKindList = legacy.TypeKindList

	// TypeKindMap identifies a map type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindMap instead.
	TypeKindMap = legacy.TypeKindMap

	// TypeKindEnum identifies a generated enum type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindEnum instead.
	TypeKindEnum = legacy.TypeKindEnum

	// TypeKindData identifies a generated structured data type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindData instead.
	TypeKindData = legacy.TypeKindData

	// TypeKindConfig identifies a generated configuration type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindConfig instead.
	TypeKindConfig = legacy.TypeKindConfig

	// TypeKindEvent identifies a generated event type.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindEvent instead.
	TypeKindEvent = legacy.TypeKindEvent

	// TypeKindTypeParameter identifies a generic type parameter.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.TypeKindTypeParameter instead.
	TypeKindTypeParameter = legacy.TypeKindTypeParameter

	// ScalarString identifies the string scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarString instead.
	ScalarString = legacy.ScalarString

	// ScalarBool identifies the boolean scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarBoolean instead.
	ScalarBool = legacy.ScalarBool

	// ScalarInt identifies the machine-sized integer scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarInt instead.
	ScalarInt = legacy.ScalarInt

	// ScalarLong identifies the 64-bit integer scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarInt instead.
	ScalarLong = legacy.ScalarLong

	// ScalarFloat identifies the 32-bit floating-point scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarFloat instead.
	ScalarFloat = legacy.ScalarFloat

	// ScalarDouble identifies the 64-bit floating-point scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarFloat instead.
	ScalarDouble = legacy.ScalarDouble

	// ScalarDecimal identifies the arbitrary-precision decimal scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarDecimal instead.
	ScalarDecimal = legacy.ScalarDecimal

	// ScalarJson identifies the arbitrary JSON scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarJSON instead.
	ScalarJson = legacy.ScalarJson

	// ScalarUuid identifies the UUID scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarUUID instead.
	ScalarUuid = legacy.ScalarUuid

	// ScalarTimestamp identifies the absolute timestamp scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarTimestamp instead.
	ScalarTimestamp = legacy.ScalarTimestamp

	// ScalarDuration identifies the duration scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarDuration instead.
	ScalarDuration = legacy.ScalarDuration

	// ScalarLocalDate identifies the local-date scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarLocalDate instead.
	ScalarLocalDate = legacy.ScalarLocalDate

	// ScalarLocalTime identifies the local-time scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarLocalTime instead.
	ScalarLocalTime = legacy.ScalarLocalTime

	// ScalarLocalDateTime identifies the local-date-time scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarLocalDateTime instead.
	ScalarLocalDateTime = legacy.ScalarLocalDateTime

	// ScalarBinary identifies the binary scalar.
	//
	// Deprecated: Use go.yorun.ai/skel/descriptor.ScalarBinary instead.
	ScalarBinary = legacy.ScalarBinary
)

// RegisterDomainSchema converts a legacy schema, validates the descriptor, and registers it in this process.
//
// Deprecated: Regenerate contracts and use RegisterDomainDescriptor instead.
func RegisterDomainSchema(schema *DomainSchema) {
	converted, err := legacy.Convert(schema)
	vpre.CheckNilError(err, "convert legacy domain schema failed")

	RegisterDomainDescriptor(converted)
}
