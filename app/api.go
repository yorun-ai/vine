package app

import (
	"reflect"
	"time"

	internalapp "go.yorun.ai/vine/internal/app"
)

// FlagApplier applies a typed runtime flag while constructing an application.
type FlagApplier = internalapp.FlagApplier

// TypeAdder adds a component or module type to an application specification.
type TypeAdder = internalapp.TypeAdder

// ListenerTypeAdder adds an event listener type with optional execution settings.
type ListenerTypeAdder = internalapp.ListenerTypeAdder

// RunnerTypeAdder adds a task runner type with optional execution settings.
type RunnerTypeAdder = internalapp.RunnerTypeAdder

// Flag marks a pointer-to-struct value as an application runtime flag.
type Flag = internalapp.Flag

// FlagModel can be embedded in a custom flag type to implement Flag.
type FlagModel = internalapp.FlagModel

// RunFlag contains the common listen address and root context for an application.
type RunFlag = internalapp.RunFlag

// App is a constructed application with start and graceful-stop lifecycle methods.
type App = internalapp.App

// ApplicationSpec describes an application's name, components, modules, and shared bindings.
type ApplicationSpec = internalapp.ApplicationSpec

// Application provides default implementations for ApplicationSpec and is intended for embedding.
type Application = internalapp.Application

// ManagedComponent is a component whose initialization, bindings, and lifecycle
// are provided by a ComponentManager. Implement it by embedding
// BaseManagedComponent and register the component through InitComponents.
type ManagedComponent = internalapp.ManagedComponent

// BaseManagedComponent associates an embedded component with its manager type M.
// Each registered component receives its own manager instance.
type BaseManagedComponent[M ComponentManager] = internalapp.BaseManagedComponent[M]

// ComponentManager initializes and manages a component. InitComponent runs
// after the component's DIInit and before consumers receive it. Component must
// return that same component; Bind publishes additional dependencies. A manager
// must not inject its own managed component; InitComponent supplies it.
type ComponentManager = internalapp.ComponentManager

// BaseComponentManager provides no-op defaults for manager hooks. Override
// InitComponent and Component to retain and return the managed component.
type BaseComponentManager = internalapp.BaseComponentManager

// ComponentLifecycle defines startup and shutdown hooks shared by modules and
// component managers. Hooks run in registration order on start and reverse order
// on stop, with components starting before modules and stopping after them.
type ComponentLifecycle = internalapp.ComponentLifecycle

// Module is a lifecycle-aware application module.
type Module = internalapp.Module

// BaseModule provides no-op lifecycle methods and is intended for embedding in modules.
type BaseModule = internalapp.BaseModule

// EventerSpec describes the event listeners exposed by an application.
type EventerSpec = internalapp.EventerSpec

// EventerEnabled provides the default event-capable application implementation.
type EventerEnabled = internalapp.EventerEnabled

// ListenerOption configures timeout, concurrency, and retry behavior for an event listener.
type ListenerOption = internalapp.ListenerOption

// ServicerSpec describes the Rpc handlers exposed by an application.
type ServicerSpec = internalapp.ServicerSpec

// ServicerEnabled provides the default Rpc-capable application implementation.
type ServicerEnabled = internalapp.ServicerEnabled

// TaskerSpec describes the task runners exposed by an application.
type TaskerSpec = internalapp.TaskerSpec

// TaskerEnabled provides the default task-capable application implementation.
type TaskerEnabled = internalapp.TaskerEnabled

// RunnerOption configures timeout, concurrency, retry, and scheduling for a task runner.
type RunnerOption = internalapp.RunnerOption

// WebberEnabled provides the default Web-capable application implementation.
type WebberEnabled = internalapp.WebberEnabled

// WebberSpec describes the Web handlers exposed by an application.
type WebberSpec = internalapp.WebberSpec

// T returns the reflection type for T without requiring a value of T.
func T[T any]() reflect.Type {
	return internalapp.T[T]()
}

// With wraps flag as an application construction option.
func With(flag Flag) FlagApplier {
	return internalapp.With(flag)
}

// WithListenerTimeout sets the maximum duration of one listener execution.
func WithListenerTimeout(timeout time.Duration) ListenerOption {
	return internalapp.WithListenerTimeout(timeout)
}

// WithListenerConcurrency sets the maximum number of concurrent listener executions.
func WithListenerConcurrency(concurrency int) ListenerOption {
	return internalapp.WithListenerConcurrency(concurrency)
}

// WithListenerNoRetry disables automatic retries for a listener.
func WithListenerNoRetry() ListenerOption {
	return internalapp.WithListenerNoRetry()
}

// WithRunnerTimeout sets the maximum duration of one task runner execution.
func WithRunnerTimeout(timeout time.Duration) RunnerOption {
	return internalapp.WithRunnerTimeout(timeout)
}

// WithRunnerConcurrency sets the maximum number of concurrent task runner executions.
func WithRunnerConcurrency(concurrency int) RunnerOption {
	return internalapp.WithRunnerConcurrency(concurrency)
}

// WithRunnerNoRetry disables automatic retries for a task runner.
func WithRunnerNoRetry() RunnerOption {
	return internalapp.WithRunnerNoRetry()
}

// WithRunnerCronScheduler schedules a runner with the named trigger Skel and cron expression.
func WithRunnerCronScheduler(triggerSkelName string, cronExpr string) RunnerOption {
	return internalapp.WithRunnerCronScheduler(triggerSkelName, cronExpr)
}
