package app

import (
	"context"
	"net/http"
	"reflect"

	coreapp "go.yorun.ai/vine/internal/core/app"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/conf"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/event"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/rpc/server"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/task"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

func (a *_AppImpl) bindRuntime(b *di.Binder) {
	b.Bind(T[runtime.App]()).ToInstance(a.info)

	for flagType, flag := range a.flags {
		b.Bind(flagType).ToFactory(func() any {
			return reflectutil.CloneStructPointer(flag)
		})
	}
	for _, configType := range conf.RegisteredTypes() {
		b.Bind(configType).ToFactory(func() conf.Config {
			return a.reader.GetByType(configType).(conf.Config)
		})
	}
}

func (a *_AppImpl) initInjector() {
	a.injector = di.NewInjector(
		a.bindRuntime,
		func(b *di.Binder) {
			b.Bind(T[context.Context]()).ToInstance(a.ctx)
			b.Bind(T[meta.Context]()).ToInstance(newMetaContext(a.ctx))
			b.BindInstance(logger.NewScopedLogger(logger.Scope{AppName: a.logicalAppName}))
		})
}

func newMetaContext(ctx context.Context) meta.Context {
	trace := meta.InitialTrace()
	initiator := meta.Initiator(nil)
	actor := meta.NewAbsentActor()
	return meta.NewContext(ctx, trace, initiator, actor)
}

func (a *_AppImpl) bindClients(b *di.Binder) {
	b.BindFactory(func(ctx meta.Context, logger *logger.Logger) *client.Client {
		return client.New(client.Option{
			Context:        ctx,
			ClientApp:      a.info,
			Logger:         logger,
			ServerEndpoint: a.linker.RpcProxyEndpoint(),
		})
	})
	for _, factory := range rpcspec.RegisteredClientFactories() {
		b.BindFactory(factory)
	}
}

func (a *_AppImpl) bindEmitters(b *di.Binder) {
	if a.isInternalApplication() {
		return
	}

	b.BindFactory(func(ctx meta.Context) *event.Emitter {
		return event.NewEmitter(event.EmitterOption{
			Context:   ctx,
			ClientApp: a.info,
			Logger: logger.NewScopedLogger(logger.Scope{
				AppName:   a.logicalAppName,
				Subsystem: logger.SubsystemEvent,
			}).With(buildLoggerFields(ctx, nil, a.info)...),
			EventClient: a.linker.EventClient(),
		})
	})
	for _, factory := range eventspec.RegisteredEventEmitterFactories() {
		b.BindFactory(factory)
	}
}

func (a *_AppImpl) bindLaunchers(b *di.Binder) {
	if a.isInternalApplication() {
		return
	}

	b.BindFactory(func(ctx meta.Context) *task.Launcher {
		return task.NewLauncher(task.LauncherOption{
			Context:   ctx,
			ClientApp: a.info,
			Logger: logger.NewScopedLogger(logger.Scope{
				AppName:   a.logicalAppName,
				Subsystem: logger.SubsystemTask,
			}).With(buildLoggerFields(ctx, nil, a.info)...),
			TaskClient: a.linker.TaskClient(),
		})
	})
	for _, factory := range taskspec.RegisteredTaskLauncherFactories() {
		b.BindFactory(factory)
	}
}

func (a *_AppImpl) initComponents() {
	componentTypes := a.componentTypes()
	checkComponentTypes(componentTypes)
	if len(componentTypes) == 0 {
		return
	}

	frameworkComponentTypes := []reflect.Type{}
	for _, componentType := range componentTypes {
		if isComponentType(componentType) {
			continue
		}
		frameworkComponentTypes = append(frameworkComponentTypes, componentType)
	}

	compMinderTypeMap := resolveFrameworkComponentMinderTypes(frameworkComponentTypes)
	minderTypes := map[reflect.Type]struct{}{}
	for _, minderType := range compMinderTypeMap {
		minderTypes[minderType] = struct{}{}
	}
	injector := a.injector.SubInjector(
		a.bindClients,
		a.bindEmitters,
		a.bindLaunchers,
		func(b *di.Binder) {
			for _, componentType := range componentTypes {
				b.Bind(componentType).In(di.SingletonScope)
			}
			for minderType := range minderTypes {
				b.Bind(minderType).In(di.TransientScope)
			}
		})

	a.frameworkComponentMinders = make([]FrameworkComponentMinder, 0, len(componentTypes))
	a.components = make([]Component, 0, len(componentTypes))
	a.componentLifecycles = make([]ComponentLifecycle, 0, len(componentTypes))
	for _, componentType := range componentTypes {
		if isComponentType(componentType) {
			component := injector.Get(componentType).Interface().(Component)
			a.components = append(a.components, component)
			a.componentLifecycles = append(a.componentLifecycles, component)
			continue
		}

		component := injector.Get(componentType).Interface().(FrameworkComponent)
		minderType := compMinderTypeMap[componentType]
		minder := injector.Get(minderType).Interface().(FrameworkComponentMinder)
		minder.InitComponent(component)
		a.frameworkComponentMinders = append(a.frameworkComponentMinders, minder)
		a.componentLifecycles = append(a.componentLifecycles, minder)
	}
}

func (a *_AppImpl) bindComponents(b *di.Binder) {
	for _, componentMinder := range a.frameworkComponentMinders {
		// Nested injectors should only see user-declared component types.
		// Framework components stay internal and may only publish extra bindings.
		b.BindInstance(componentMinder.Component())
		componentMinder.Bind(b)
	}
	for _, component := range a.components {
		b.BindInstance(component)
		component.Bind(b)
	}
}

func (a *_AppImpl) bindCommon(b *di.Binder) {
	a.spec.BindCommon(b)
}

func (a *_AppImpl) initModules() {
	moduleTypes := a.moduleTypes()
	checkTypes[Module]("module", moduleTypes)
	if len(moduleTypes) == 0 {
		return
	}

	injector := a.injector.SubInjector(
		a.bindClients,
		a.bindEmitters,
		a.bindLaunchers,
		a.bindComponents,
		a.bindCommon,
		func(b *di.Binder) {
			for _, moduleType := range moduleTypes {
				b.Bind(moduleType).In(di.SingletonScope)
			}
		})
	a.modules = make([]Module, 0, len(moduleTypes))
	for _, moduleType := range moduleTypes {
		module := injector.Get(moduleType).Interface().(Module)
		a.modules = append(a.modules, module)
	}
}

func (a *_AppImpl) componentTypes() []reflect.Type {
	var componentTypes []reflect.Type
	a.spec.InitComponents(func(componentType reflect.Type) {
		componentTypes = append(componentTypes, componentType)
	})
	return componentTypes
}

func (a *_AppImpl) moduleTypes() []reflect.Type {
	var moduleTypes []reflect.Type
	a.spec.InitModules(func(moduleType reflect.Type) {
		moduleTypes = append(moduleTypes, moduleType)
	})
	return moduleTypes
}

func (a *_AppImpl) bindModules(b *di.Binder) {
	for _, module := range a.modules {
		// Framework-owned module instances are always rebound here for nested
		// injectors such as servicer execution containers. Module.Bind should
		// only publish extra bindings and must not re-bind the module itself.
		b.BindInstance(module)
		module.Bind(b)
	}
}

func (a *_AppImpl) bindAppDeps(b *di.Binder) {
	a.bindRuntime(b)
	a.bindComponents(b)
	a.bindClients(b)
	a.bindEmitters(b)
	a.bindLaunchers(b)
	a.bindCommon(b)
	a.bindModules(b)
}

func checkTypes[I any](kind string, types []reflect.Type) {
	targetType := T[I]()
	seen := map[reflect.Type]struct{}{}
	for _, defType := range types {
		vpre.Check(defType != nil, "%s type must not be nil", kind)
		vpre.Check(reflectutil.IsStructPointerType(defType), "%s type %s must be pointer to struct", kind, defType)
		vpre.Check(defType.Implements(targetType), "%s type %s must implement %s", kind, defType, targetType)
		_, exists := seen[defType]
		vpre.Check(!exists, "%s type %s already declared", kind, defType)
		seen[defType] = struct{}{}
	}
}

func checkComponentTypes(componentTypes []reflect.Type) {
	seen := map[reflect.Type]struct{}{}
	for _, componentType := range componentTypes {
		vpre.Check(componentType != nil, "component type must not be nil")
		vpre.Check(reflectutil.IsStructPointerType(componentType), "component type %s must be pointer to struct", componentType)
		isComponent := isComponentType(componentType)
		_, exists := seen[componentType]
		vpre.Check(!exists, "component type %s already declared", componentType)
		seen[componentType] = struct{}{}
		if isComponent {
			continue
		}
		vpre.Check(componentType.Implements(T[FrameworkComponent]()),
			"component type %s must implement %s or %s", componentType, T[Component](), T[FrameworkComponent]())
	}
}

func (a *_AppImpl) initServers() {
	logger.FreezePayloadPolicies()
	if a.shouldEnableConsole() {
		a.consoleServer = server.New(server.Option{
			App:            a.info,
			LogicalAppName: a.logicalAppName,
			MuteVerboseLog: true,
			HandlerTypes:   []reflect.Type{T[*ConsoleServiceServerImpl]()},
		})
		a.appendRoute(coreapp.PathConsole, a.consoleServer.HTTPHandler(), a.consoleServer.RpcHandler())
	}

	if servicerSpec, ok := a.spec.(ServicerSpec); ok {
		a.servicer = newServicer(servicerSpec, a.info, a.bindAppDeps, a.logicalAppName)
		a.appendRoute(coreapp.PathRpcInvoke, a.servicer.httpHandler(), a.servicer.rpcHandler())
	}

	if webberSpec, ok := a.spec.(WebberSpec); ok {
		a.webber = newWebber(webberSpec, a.info, a.bindAppDeps, a.logicalAppName)
		a.appendRoute(coreapp.PathWebAccess, a.webber.httpHandler(), nil)
	}

	if eventerSpec, ok := a.spec.(EventerSpec); ok {
		a.eventer = newEventer(eventerSpec, a.info, a.bindAppDeps, a.logicalAppName)
		a.appendRoute(coreapp.PathEvent, a.eventer.httpHandler(), a.eventer.rpcHandler())
	}

	if taskerSpec, ok := a.spec.(TaskerSpec); ok {
		a.tasker = newTasker(taskerSpec, a.info, a.bindAppDeps, a.logicalAppName)
		a.appendRoute(coreapp.PathTask, a.tasker.httpHandler(), a.tasker.rpcHandler())
	}

	for _, module := range a.modules {
		if routeModule, ok := module.(PathPrefixRouteModule); ok {
			routeModule.InitPathPrefixRoute(func(prefix string, httpHandler http.Handler, rpcHandler rpcspec.RpcHandler) {
				a.appendRoute(prefix, httpHandler, rpcHandler)
			})
		}
	}
}

type ConsoleServiceServerImpl struct {
	appskeled.DefaultConsoleServiceServer
}

func (*ConsoleServiceServerImpl) Ping() {}
