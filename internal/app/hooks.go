package app

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
)

// HookAdder registers initialization callbacks in execution order.
// Callbacks must be non-variadic functions without return values.
// Their parameters are resolved from the current initialization stage's injector.
type HookAdder struct {
	afterBootstrap  []any
	afterComponents []any
	afterModules    []any
}

// AfterAppBootstrap registers a callback after base dependencies and clients are available,
// before any components are initialized.
func (a *HookAdder) AfterAppBootstrap(callback any) {
	checkInitHook(callback)
	a.afterBootstrap = append(a.afterBootstrap, callback)
}

func (a *_AppImpl) afterAppBootstrap() {
	if len(a.hooks.afterBootstrap) == 0 {
		return
	}
	hookInjector := a.injector.SubInjector(
		a.bindClients,
		a.bindEmitters,
		a.bindLaunchers,
	)
	invokeInitHooks(hookInjector, a.hooks.afterBootstrap)
}

// AfterComponentsInitialized registers a callback after all components are initialized.
func (a *HookAdder) AfterComponentsInitialized(callback any) {
	checkInitHook(callback)
	a.afterComponents = append(a.afterComponents, callback)
}

// AfterModulesInitialized registers a callback after all modules are initialized.
func (a *HookAdder) AfterModulesInitialized(callback any) {
	checkInitHook(callback)
	a.afterModules = append(a.afterModules, callback)
}

func checkInitHook(callback any) {
	value := reflect.ValueOf(callback)
	vpre.Check(value.IsValid() && value.Kind() == reflect.Func, "initialization hook must be a function")
	vpre.Check(!value.IsNil(), "initialization hook must not be nil")
	kind := value.Type()
	vpre.Check(!kind.IsVariadic() && kind.NumOut() == 0, "initialization hook must be non-variadic and have no return values")
}

func invokeInitHooks(injector di.PlainInjector, callbacks []any) {
	for _, callback := range callbacks {
		injector.Invoke(callback)
	}
}
