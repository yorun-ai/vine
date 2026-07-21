package app

import (
	"context"
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

// Flag

type Flag interface {
	mustBeFlag()
}

type FlagModel struct{}

func (FlagModel) mustBeFlag() {}

// RunFlag

type RunFlag struct {
	FlagModel
	ListenAddr   string
	LinkEndpoint string
	Context      context.Context
}

// Flags

type _Flags map[reflect.Type]Flag

func (f _Flags) EnsureRunFlag() {
	if _, ok := f[T[*RunFlag]()].(*RunFlag); !ok {
		f[T[*RunFlag]()] = &RunFlag{}
	}
}

func (f _Flags) ListenAddr() string {
	return f[T[*RunFlag]()].(*RunFlag).ListenAddr
}

func (f _Flags) LinkEndpoint() string {
	return f[T[*RunFlag]()].(*RunFlag).LinkEndpoint
}

func (f _Flags) Context() context.Context {
	ctx := f[T[*RunFlag]()].(*RunFlag).Context
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// FlagApplier

type FlagApplier func(_Flags)

func (f _Flags) Apply(appliers ...FlagApplier) {
	for _, applier := range appliers {
		if applier != nil {
			applier(f)
		}
	}
}

func With(flag Flag) FlagApplier {
	kind := reflect.TypeOf(flag)
	vpre.Check(kind != nil, "flag must not be nil")
	vpre.Check(reflectutil.IsStructPointerType(kind), "flag %s must be pointer to struct", kind)
	return func(flags _Flags) {
		_, exists := flags[kind]
		vpre.Check(!exists, "type %s was already provided", kind)
		flags[kind] = flag
	}
}

func WithLinkEndpoint(linkEndpoint string) FlagApplier {
	return func(flags _Flags) {
		flags.EnsureRunFlag()
		flags[T[*RunFlag]()].(*RunFlag).LinkEndpoint = linkEndpoint
	}
}
