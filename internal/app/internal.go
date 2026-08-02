package app

import (
	"net/http"
	"reflect"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
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
	BackendIdentity   *mtls.Identity
	RPCTransport      http.RoundTripper
	DisableConsole    bool
	DisableHTTPServer bool
	ProtectHTTPServer bool
	HTTPServerClients []string

	InprocHostPath string
}

func (a *InternalApplication) internalAttrs() *InternalAttributes {
	return &a.InternalAttrs
}

// InternalInproc

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

// InternalRuntime

// InternalRuntime exposes runtime-only capabilities to modules belonging to an
// internal Vine application. It is not bound for ordinary applications.
type InternalRuntime interface {
	// AdditionalServicer creates Rpc transport handlers that reuse the owning
	// application's runtime dependency graph and serve only the requested
	// handler types. It must only be called after module construction, such as
	// from BeforeAppStart; DIInit runs before all modules have been assembled.
	AdditionalServicer(handlerTypes ...reflect.Type) (http.Handler, rpcspec.RpcHandler)
}

func (a *_AppImpl) AdditionalServicer(handlerTypes ...reflect.Type) (http.Handler, rpcspec.RpcHandler) {
	servicer := &_Servicer{
		appInfo:      a.info,
		handlerTypes: append([]reflect.Type(nil), handlerTypes...),
		bindAppDeps:  a.bindAppDeps,
	}
	servicer.init()
	return servicer.httpHandler(), servicer.rpcHandler()
}
