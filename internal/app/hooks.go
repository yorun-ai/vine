package app

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
)

// HookAdder registers injected application lifecycle callbacks.
// Startup callbacks run in registration order; shutdown callbacks run in reverse order.
type HookAdder struct {
	beforeStart []_AppHook
	afterStart  []_AppHook
	beforeStop  []_AppHook
	afterStop   []_AppHook
}

type _AppHook struct {
	function  reflect.Value
	arguments []reflect.Value
}

// BeforeAppStart runs before component and module pre-start callbacks.
// The callback may return an error to abort startup.
func (a *HookAdder) BeforeAppStart(callback any) {
	a.beforeStart = append(a.beforeStart, newAppHook(callback, true))
}

// AfterAppStart runs after registration and all component and module post-start callbacks.
func (a *HookAdder) AfterAppStart(callback any) {
	a.afterStart = append(a.afterStart, newAppHook(callback, false))
}

// BeforeAppStop runs before module and component pre-stop callbacks.
func (a *HookAdder) BeforeAppStop(callback any) {
	a.beforeStop = append(a.beforeStop, newAppHook(callback, false))
}

// AfterAppStop runs after module and component post-stop callbacks.
// Injected resources may already be closed and the application context is canceled.
func (a *HookAdder) AfterAppStop(callback any) {
	a.afterStop = append(a.afterStop, newAppHook(callback, false))
}

func newAppHook(callback any, allowError bool) _AppHook {
	value := reflect.ValueOf(callback)
	vpre.Check(value.IsValid() && value.Kind() == reflect.Func, "application hook must be a function")
	vpre.Check(!value.IsNil(), "application hook must not be nil")
	kind := value.Type()
	vpre.Check(!kind.IsVariadic(), "application hook must be non-variadic")
	validResult := kind.NumOut() == 0 || (allowError && kind.NumOut() == 1 && kind.Out(0) == T[error]())
	vpre.Check(validResult, "application hook has invalid return values; only BeforeAppStart may return error")
	return _AppHook{function: value}
}

func (a *_AppImpl) prepareHooks() {
	groups := [][]_AppHook{a.hooks.beforeStart, a.hooks.afterStart, a.hooks.beforeStop, a.hooks.afterStop}
	if len(a.hooks.beforeStart)+len(a.hooks.afterStart)+len(a.hooks.beforeStop)+len(a.hooks.afterStop) == 0 {
		return
	}
	injector := a.moduleInjector
	if injector == nil {
		injector = a.injector.SubInjector(
			a.bindClients,
			a.bindEmitters,
			a.bindLaunchers,
			a.bindComponents,
			a.bindCommon,
		)
	}
	injector = injector.SubInjector(func(b *di.Binder) {
		for _, module := range a.modules {
			module.Bind(b)
		}
	})

	for _, hooks := range groups {
		for i := range hooks {
			hook := &hooks[i]
			for parameter := range hook.function.Type().Ins() {
				hook.arguments = append(hook.arguments, injector.Get(parameter))
			}
		}
	}
}

func invokeHooks(hooks []_AppHook, reverse bool) error {
	for i := range hooks {
		if reverse {
			i = len(hooks) - 1 - i
		}
		hook := hooks[i]
		result := hook.function.Call(hook.arguments)
		if len(result) == 1 && !result[0].IsNil() {
			return result[0].Interface().(error)
		}
	}
	return nil
}
