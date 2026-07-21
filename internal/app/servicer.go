package app

import (
	"context"
	"net/http"
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcserver "go.yorun.ai/vine/internal/core/rpc/server"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/runtime"
)

type ServicerSpec interface {
	ServicerBind(b *di.Binder)
	ServicerInitHandlers(addHandler TypeAdder)
	ServicerInitFilters(addFilter TypeAdder)
}

type ServicerEnabled struct{}

func (*ServicerEnabled) ServicerBind(*di.Binder)                   {}
func (*ServicerEnabled) ServicerInitHandlers(addHandler TypeAdder) {}
func (*ServicerEnabled) ServicerInitFilters(addFilter TypeAdder)   {}

type _Servicer struct {
	spec        ServicerSpec
	appInfo     runtime.App
	bindAppDeps di.BindApplier

	server *rpcserver.Server
}

func newServicer(spec ServicerSpec, info runtime.App, deps di.BindApplier) *_Servicer {
	servicer := &_Servicer{
		spec:        spec,
		appInfo:     info,
		bindAppDeps: deps,
	}
	servicer.init()
	return servicer
}

func (s *_Servicer) init() {
	bindAppliers := []di.BindApplier{
		s.bindAppDeps,
		s.bindContext,
		s.bindLogger,
		s.spec.ServicerBind,
	}

	s.server = rpcserver.New(rpcserver.Option{
		App:          s.appInfo,
		HandlerTypes: s.handlerTypes(),
		Executor:     rpcserver.NewContainerExecutor(s.filterTypes(), bindAppliers),
	})
}

func (s *_Servicer) handlerTypes() []reflect.Type {
	var handlerTypes []reflect.Type
	s.spec.ServicerInitHandlers(func(handlerType reflect.Type) {
		handlerTypes = append(handlerTypes, handlerType)
	})
	return handlerTypes
}

func (s *_Servicer) filterTypes() []reflect.Type {
	var filterTypes []reflect.Type
	s.spec.ServicerInitFilters(func(filterType reflect.Type) {
		filterTypes = append(filterTypes, filterType)
	})
	return filterTypes
}

func (*_Servicer) bindContext(b *di.Binder) {
	b.BindFactory(func(ctx rpcspec.Context) context.Context {
		return ctx
	})
	b.BindFactory(func(ctx rpcspec.Context) meta.Context {
		return ctx
	})
	bindActor(b)
}

func (s *_Servicer) bindLogger(b *di.Binder) {
	b.BindFactory(func(ctx rpcspec.Context, method rpcspec.MethodInfo) *logger.Logger {
		return logger.NewLogger(logger.GlobalOption()).With(buildLoggerFields(ctx, method, s.appInfo)...)
	})
}

func (s *_Servicer) rpcHandler() rpcspec.RpcHandler {
	return s.server.RpcHandler()
}

func (s *_Servicer) httpHandler() http.Handler {
	return s.server.HTTPHandler()
}

func (s *_Servicer) serviceHandlerRegistrations() []linkskeled.ServiceHandlerRegistration {
	serviceInfos := s.server.GetServiceInfos()
	registrations := make([]linkskeled.ServiceHandlerRegistration, 0, len(serviceInfos))
	for _, serviceInfo := range serviceInfos {
		registrations = append(registrations, linkskeled.ServiceHandlerRegistration{
			ServiceSkelName: serviceInfo.SkelName(),
			SchemaHash:      serviceInfo.Hash(),
		})
	}
	return registrations
}

func (s *_Servicer) start() {}
