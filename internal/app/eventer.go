package app

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/event"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcserver "go.yorun.ai/vine/internal/core/rpc/server"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/runtime"
)

type EventerSpec interface {
	EventerBind(b *di.Binder)
	EventerInitListeners(addListener ListenerTypeAdder)
	EventerInitFilters(addFilter TypeAdder)
}

type EventerEnabled struct{}

func (*EventerEnabled) EventerBind(*di.Binder)                 {}
func (*EventerEnabled) EventerInitListeners(ListenerTypeAdder) {}
func (*EventerEnabled) EventerInitFilters(TypeAdder)           {}

type _Eventer struct {
	spec        EventerSpec
	appInfo     runtime.App
	appName     string
	bindAppDeps di.BindApplier

	listeners   []_ListenerTypeEntry
	eventServer *event.Server
	rpcServer   *rpcserver.Server
}

func newEventer(spec EventerSpec, info runtime.App, deps di.BindApplier, logicalAppNames ...string) *_Eventer {
	appName := info.Name()
	if len(logicalAppNames) > 0 {
		appName = logicalAppNames[0]
	}
	eventer := &_Eventer{
		spec:        spec,
		appInfo:     info,
		appName:     appName,
		bindAppDeps: deps,
	}
	eventer.init()
	return eventer
}

func (e *_Eventer) init() {
	bindAppliers := []di.BindApplier{
		e.bindAppDeps,
		e.bindContext,
		e.bindLogger,
		e.spec.EventerBind,
	}

	e.listeners = e.collectListeners()
	e.eventServer = event.NewServer(event.Option{
		App:               e.appInfo,
		LogicalAppName:    e.appName,
		ListenerImplTypes: e.listenerTypes(),
		Executor:          event.NewContainerExecutor(e.filterTypes(), bindAppliers),
	})

	e.rpcServer = rpcserver.New(rpcserver.Option{
		App:            e.appInfo,
		LogicalAppName: e.appName,
		HandlerTypes:   []reflect.Type{T[*_AppEventServiceServerImpl]()},
		Executor:       rpcserver.NewDefaultExecutor(rpcserver.With(e.eventServer)),
	})
}

func (e *_Eventer) listenerTypes() []reflect.Type {
	var listenerTypes []reflect.Type
	for _, listenerEntry := range e.listeners {
		listenerTypes = append(listenerTypes, listenerEntry.kind)
	}
	return listenerTypes
}

func (e *_Eventer) collectListeners() []_ListenerTypeEntry {
	var listenerEntries []_ListenerTypeEntry
	e.spec.EventerInitListeners(func(listenerType reflect.Type, options ...ListenerOption) {
		listenerEntries = append(listenerEntries, _ListenerTypeEntry{
			kind:    listenerType,
			options: newListenerOptions(options),
		})
	})
	return listenerEntries
}

func (e *_Eventer) filterTypes() []reflect.Type {
	var filterTypes []reflect.Type
	e.spec.EventerInitFilters(func(filterType reflect.Type) {
		filterTypes = append(filterTypes, filterType)
	})
	return filterTypes
}

func (*_Eventer) bindContext(b *di.Binder) {
	b.BindFactory(func(ctx eventspec.Context) context.Context {
		return ctx
	})
	b.BindFactory(func(ctx eventspec.Context) meta.Context {
		return ctx
	})
}

func (e *_Eventer) bindLogger(b *di.Binder) {
	b.BindFactory(func(ctx eventspec.Context, eventInfo eventspec.EventInfo) *logger.Logger {
		fields := buildContextLogFields(ctx)
		if eventInfo != nil {
			fields = append(fields,
				slog.String("eventName", eventInfo.Name()),
				slog.String("eventSkelName", eventInfo.SkelName()),
			)
		}
		if e.appInfo != nil {
			fields = append(fields,
				slog.String("name", e.appInfo.Name()),
				slog.String("version", e.appInfo.Version()),
				slog.String("instanceId", e.appInfo.InstanceId()),
			)
		}
		return logger.NewScopedLogger(logger.Scope{AppName: e.appName}).With(fields...)
	})
}

func (e *_Eventer) httpHandler() http.Handler {
	return e.rpcServer.HTTPHandler()
}

func (e *_Eventer) rpcHandler() rpcspec.RpcHandler {
	return e.rpcServer.RpcHandler()
}

func (e *_Eventer) eventListenerRegistrations() []linkskeled.EventListenerRegistration {
	listenerImplDict := eventspec.NewListenerImplDict()
	infoByType := map[reflect.Type]eventspec.EventInfo{}
	for _, listenerEntry := range e.listeners {
		listenerImplDict.Add(listenerEntry.kind)
	}
	listenerImplDict.IterateListenerImpl(func(listenerImpl eventspec.ListenerImpl) {
		infoByType[listenerImpl.Type()] = listenerImpl.Info()
	})

	registrations := make([]linkskeled.EventListenerRegistration, 0, len(e.listeners))
	for _, listenerEntry := range e.listeners {
		eventInfo := infoByType[listenerEntry.kind]
		registrations = append(registrations, linkskeled.EventListenerRegistration{
			EventSkelName: eventInfo.SkelName(),
			SchemaHash:    eventInfo.Hash(),
			TimeoutMs:     int(listenerEntry.options.Timeout / time.Millisecond),
			Concurrency:   listenerEntry.options.Concurrency,
			NoRetry:       listenerEntry.options.NoRetry,
		})
	}
	return registrations
}

func (*_Eventer) start() {}

type _AppEventServiceServerImpl struct {
	appskeled.DefaultEventServiceServer

	Context     rpcspec.Context
	EventServer *event.Server
}

func (s *_AppEventServiceServerImpl) OnEvent(on appskeled.EventOn) {
	onErr := s.EventServer.OnEvent(s.Context, on)
	ex.PanicIfError(onErr)
}
