package meta

import (
	"context"

	internalmeta "go.yorun.ai/vine/internal/core/meta"
)

// App identifies one running application instance.
type App = internalmeta.App

// Context combines a Go context with Vine trace, initiator, and actor metadata.
type Context = internalmeta.Context

// Initiator identifies the application and network peer that originated a call chain.
type Initiator = internalmeta.Initiator

// Actor represents the identity on whose behalf a call executes.
type Actor = internalmeta.Actor

// ActorSpec describes the encoding and type of a registered actor payload.
type ActorSpec = internalmeta.ActorSpec

// ActorType classifies absent, anonymous, authenticated, and impersonated actors.
type ActorType = internalmeta.ActorType

// Trace identifies a distributed trace and its current span.
type Trace = internalmeta.Trace

const (
	// ActorTypeAbsent indicates that no actor identity was supplied.
	ActorTypeAbsent = internalmeta.ActorTypeAbsent
	// ActorTypeAnonymous identifies an unauthenticated external actor.
	ActorTypeAnonymous = internalmeta.ActorTypeAnonymous
	// ActorTypeAuthenticated identifies a verified actor.
	ActorTypeAuthenticated = internalmeta.ActorTypeAuthenticated
	// ActorTypeImpersonated identifies an actor operating as another identity.
	ActorTypeImpersonated = internalmeta.ActorTypeImpersonated
)

// NewApp validates and creates an application identity.
func NewApp(name string, version string, instanceId string) (App, error) {
	return internalmeta.NewApp(name, version, instanceId)
}

// DecodeAppFromDelimited decodes an application identity from its delimited wire form.
func DecodeAppFromDelimited(value string) (App, error) {
	return internalmeta.DecodeAppFromDelimited(value)
}

// EncodeAppToDelimited encodes app into its delimited wire form.
func EncodeAppToDelimited(app App) string {
	return internalmeta.EncodeAppToDelimited(app)
}

// MustNewApp is like NewApp but panics when the identity is invalid.
func MustNewApp(name string, version string, instanceId string) App {
	return internalmeta.MustNewApp(name, version, instanceId)
}

// MustNewAppWithRandomId creates an application identity with a random instance ID.
func MustNewAppWithRandomId(name string, version string) App {
	return internalmeta.MustNewAppWithRandomId(name, version)
}

// IsValidName reports whether name is a valid Vine application name.
func IsValidName(name string) bool {
	return internalmeta.IsValidName(name)
}

// IsValidVersion reports whether version is accepted as application metadata.
func IsValidVersion(version string) bool {
	return internalmeta.IsValidVersion(version)
}

// IsValidInstanceId reports whether instanceId is a valid application instance identifier.
func IsValidInstanceId(instanceId string) bool {
	return internalmeta.IsValidInstanceId(instanceId)
}

// NewContext combines ctx with Vine call metadata.
func NewContext(ctx context.Context, trace Trace, initiator Initiator, actor Actor) Context {
	return internalmeta.NewContext(ctx, trace, initiator, actor)
}

// NewInitiator validates and creates call-origin metadata.
func NewInitiator(name string, version string, instanceId string, dialer string, ipStr string) (Initiator, error) {
	return internalmeta.NewInitiator(name, version, instanceId, dialer, ipStr)
}

// NewAbsentActor creates an actor representing the absence of an identity.
func NewAbsentActor() Actor {
	return internalmeta.NewAbsentActor()
}

// NewAnonymousActor creates an unauthenticated actor.
func NewAnonymousActor() Actor {
	return internalmeta.NewAnonymousActor()
}

// NewAuthenticatedActor creates an authenticated actor carrying info.
func NewAuthenticatedActor[I any](info I) Actor {
	return internalmeta.NewAuthenticatedActor[I](info)
}

// GetActorInfo returns actor's payload as T and reports whether the conversion succeeded.
func GetActorInfo[T any](actor Actor) (T, bool) {
	return internalmeta.GetActorInfo[T](actor)
}

// MustGetActorInfo returns actor's payload as T or panics when it has another type.
func MustGetActorInfo[T any](actor Actor) T {
	return internalmeta.MustGetActorInfo[T](actor)
}

// RegisterActor adds spec to the process-wide actor registry.
func RegisterActor(spec ActorSpec) {
	internalmeta.RegisterActor(spec)
}

// DecodeActorFromBase64 decodes an actor from its Base64 wire representation.
func DecodeActorFromBase64(value string) (Actor, error) {
	return internalmeta.DecodeActorFromBase64(value)
}

// EncodeActorToBase64 encodes actor into its Base64 wire representation.
func EncodeActorToBase64(actor Actor) string {
	return internalmeta.EncodeActorToBase64(actor)
}

// NewId returns a new trace identifier.
func NewId() string {
	return internalmeta.NewId()
}

// NewSpan returns a new span identifier.
func NewSpan() string {
	return internalmeta.NewSpan()
}

// IsValidSpan reports whether span is a valid span identifier.
func IsValidSpan(span string) bool {
	return internalmeta.IsValidSpan(span)
}

// NewTrace validates and creates a trace from id and span.
func NewTrace(id string, span string) (Trace, error) {
	return internalmeta.NewTrace(id, span)
}

// DecodeTraceFromDelimited decodes a trace from its delimited wire form.
func DecodeTraceFromDelimited(value string) (Trace, error) {
	return internalmeta.DecodeTraceFromDelimited(value)
}

// DecodeTraceFromDelimitedOrNewSpan decodes a trace and replaces or creates its current span.
func DecodeTraceFromDelimitedOrNewSpan(value string) (Trace, error) {
	return internalmeta.DecodeTraceFromDelimitedOrNewSpan(value)
}

// EncodeTraceToDelimited encodes trace into its delimited wire form.
func EncodeTraceToDelimited(trace Trace) string {
	return internalmeta.EncodeTraceToDelimited(trace)
}

// InitialTrace creates a new trace with an initial span.
func InitialTrace() Trace {
	return internalmeta.InitialTrace()
}

// IsValidId reports whether id is a valid trace identifier.
func IsValidId(id string) bool {
	return internalmeta.IsValidId(id)
}

// DecodeInitiatorFromBase64 decodes initiator metadata from its Base64 wire representation.
func DecodeInitiatorFromBase64(value string) (Initiator, error) {
	return internalmeta.DecodeInitiatorFromBase64(value)
}

// EncodeInitiatorToBase64 encodes initiator metadata into its Base64 wire representation.
func EncodeInitiatorToBase64(initiator Initiator) string {
	return internalmeta.EncodeInitiatorToBase64(initiator)
}
