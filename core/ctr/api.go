package ctr

import (
	internalctr "go.yorun.ai/vine/internal/core/ctr"
)

type (
	// Option configures a Container's injector and filters.
	Option = internalctr.Option
	// Container creates dependency-injected method executions.
	Container = internalctr.Container
	// Execution is a prepared method invocation.
	Execution = internalctr.Execution
	// Context carries state for one container execution.
	Context = internalctr.Context
	// FilterNext invokes the next filter or target method in a chain.
	FilterNext = internalctr.FilterNext
	// Filter intercepts a container execution.
	Filter = internalctr.Filter
)

// NewContainer creates a method execution container from option.
func NewContainer(option Option) Container {
	return internalctr.NewContainer(option)
}
