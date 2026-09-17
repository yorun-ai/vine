package meta

// CurrentApp is the identity of the application instance that owns the
// component graph it is injected into. It carries the same fields as App but is
// a distinct type, so the binding for the current application cannot collide
// with App values that describe other applications, such as callers, callees,
// and registered peers.
//
// It marks the owning application itself: the dependency injection binding, the
// fields that hold that binding, and the constructors that build a component for
// it. APIs that accept any application identity, such as a client that calls
// another application on behalf of a caller, keep using App.
type CurrentApp interface {
	Name() string
	Version() string
	InstanceId() string
}
