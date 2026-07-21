package event

import (
	"reflect"

	"go.yorun.ai/vine/core/di"
	internalevent "go.yorun.ai/vine/internal/core/event"
	"go.yorun.ai/vine/internal/core/event/spec"
)

// EmitterOption configures an event emitter and its transport.
type EmitterOption = internalevent.EmitterOption

// EmitOption configures one event emission.
type EmitOption = internalevent.EmitOption

// Emitter publishes registered events.
type Emitter = internalevent.Emitter

// ServerOption configures an event listener server.
type ServerOption = internalevent.Option

// Executor invokes an event listener.
type Executor = internalevent.Executor

// Server receives events and dispatches them to an Executor.
type Server = internalevent.Server

// Context carries metadata for one event listener execution.
type Context = spec.Context

// On is the transport envelope for an event delivery.
type On = spec.On

// EventSpecType describes whether an event is emitted, listened to, or both.
type EventSpecType = spec.EventSpecType

// EventSpec describes a generated event contract.
type EventSpec = spec.EventSpec

// EventInfo is the runtime metadata derived from an EventSpec.
type EventInfo = spec.EventInfo

const (
	// EventSpecTypeListener identifies a listener-only event contract.
	EventSpecTypeListener = spec.EventSpecTypeListener
	// EventSpecTypeEmitter identifies an emitter-only event contract.
	EventSpecTypeEmitter = spec.EventSpecTypeEmitter
	// EventSpecTypeBoth identifies a contract that can emit and listen.
	EventSpecTypeBoth = spec.EventSpecTypeBoth
)

// NewEmitter creates an event emitter from option.
func NewEmitter(option EmitterOption) *Emitter {
	return internalevent.NewEmitter(option)
}

// NewServer creates an event listener server from option.
func NewServer(option ServerOption) *Server {
	return internalevent.NewServer(option)
}

// NewContainerExecutor creates an Executor backed by a DI container and filter chain.
func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return internalevent.NewContainerExecutor(filterTypes, bindAppliers)
}

// Register adds eventSpec to the process-wide event registry.
func Register(eventSpec *EventSpec) {
	spec.Register(eventSpec)
}
