package app

import (
	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/util/vpre"
)

// InternalApplication

type InternalApplicationSpec interface {
	internalAttrs() *InternalAttributes
}

type InternalApplication struct {
	Application

	InternalAttrs InternalAttributes
}

type InternalAttributes struct {
	Info              runtime.App
	Linker            link.Linker
	DisableConsole    bool
	DisableHTTPServer bool

	InprocHostPath string
}

func (a *InternalApplication) internalAttrs() *InternalAttributes {
	return &a.InternalAttrs
}

type InternalInprocFlag struct {
	FlagModel

	// Enabled is safe to use during spec initialization. Other fields may still
	// be finalized later by app construction, so only Enabled should be
	// considered accurate inside specs.
	Enabled  bool
	HostPath string
}

func (f _Flags) InitInprocFlag(enableInproc bool) {
	f[T[*InternalInprocFlag]()] = &InternalInprocFlag{
		Enabled: enableInproc,
	}
}

func (f _Flags) InprocFlag() *InternalInprocFlag {
	flag, ok := f[T[*InternalInprocFlag]()].(*InternalInprocFlag)
	vpre.Check(ok, "inproc flag missing")
	return flag
}
