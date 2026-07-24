package app

import (
	"context"
	"net/http"
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/runtime"
	webserver "go.yorun.ai/vine/internal/core/web/server"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
)

type WebberSpec interface {
	WebberBind(b *di.Binder)
	WebberInitHandlers(addHandler TypeAdder)
	WebberInitFilters(addFilter TypeAdder)
}

type WebberEnabled struct{}

func (*WebberEnabled) WebberBind(*di.Binder)                   {}
func (*WebberEnabled) WebberInitHandlers(addHandler TypeAdder) {}
func (*WebberEnabled) WebberInitFilters(addFilter TypeAdder)   {}

type _Webber struct {
	spec        WebberSpec
	appInfo     runtime.App
	appName     string
	bindAppDeps di.BindApplier

	server *webserver.Server
}

func newWebber(spec WebberSpec, info runtime.App, deps di.BindApplier, logicalAppNames ...string) *_Webber {
	appName := info.Name()
	if len(logicalAppNames) > 0 {
		appName = logicalAppNames[0]
	}
	webber := &_Webber{
		spec:        spec,
		appInfo:     info,
		appName:     appName,
		bindAppDeps: deps,
	}
	webber.init()
	return webber
}

func (w *_Webber) init() {
	bindAppliers := []di.BindApplier{
		w.bindAppDeps,
		w.bindContext,
		w.bindLogger,
		w.spec.WebberBind,
	}

	w.server = webserver.NewServer(webserver.Option{
		HandlerTypes: w.handlerTypes(),
		Executor:     webserver.NewContainerExecutor(w.filterTypes(), bindAppliers),
	})
}

func (w *_Webber) handlerTypes() []reflect.Type {
	var handlerTypes []reflect.Type
	w.spec.WebberInitHandlers(func(handlerType reflect.Type) {
		handlerTypes = append(handlerTypes, handlerType)
	})
	return handlerTypes
}

func (w *_Webber) filterTypes() []reflect.Type {
	var filterTypes []reflect.Type
	w.spec.WebberInitFilters(func(filterType reflect.Type) {
		filterTypes = append(filterTypes, filterType)
	})
	return filterTypes
}

func (*_Webber) bindContext(b *di.Binder) {
	b.BindFactory(func(ctx webspec.Context) context.Context {
		return ctx
	})
	b.BindFactory(func(ctx webspec.Context) meta.Context {
		return ctx
	})
	bindActor(b)
}

func (w *_Webber) bindLogger(b *di.Binder) {
	b.BindFactory(func(ctx meta.Context) *logger.Logger {
		return logger.NewScopedLogger(logger.Scope{AppName: w.appName}).With(buildLoggerFields(ctx, nil, w.appInfo)...)
	})
}

func (w *_Webber) httpHandler() http.Handler {
	return w.server.HTTPHandler()
}
