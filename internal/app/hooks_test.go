package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
)

type initHooksSpec struct {
	Application
	empty    bool
	register func(*HookAdder)
}

func (*initHooksSpec) Name() string               { return "test.init.hooks" }
func (s *initHooksSpec) InitHooks(add *HookAdder) { s.register(add) }
func (s *initHooksSpec) InitComponents(add TypeAdder) {
	if !s.empty {
		add(T[*initHooksComponent]())
	}
}
func (s *initHooksSpec) InitModules(add TypeAdder) {
	if !s.empty {
		add(T[*initHooksModule]())
	}
}

type initHooksCommon struct {
	di.SingletonScoped
	Component *initHooksComponent `inject:""`
}

func (s *initHooksSpec) BindCommon(b *di.Binder) {
	if !s.empty {
		b.Bind(T[*initHooksCommon]()).In(di.SingletonScope)
	}
}

type initHooksComponent struct {
	BaseComponent
	Ready bool
}

func (c *initHooksComponent) DIInit() { c.Ready = true }

type initHooksModule struct {
	BaseModule
	Component *initHooksComponent `inject:""`
	Common    *initHooksCommon    `inject:""`
	Ready     bool
}

func (m *initHooksModule) DIInit() { m.Ready = true }

func TestInitializationHooks(t *testing.T) {
	events := []string{}
	var component *initHooksComponent
	var module *initHooksModule
	var a *_AppImpl
	spec := &initHooksSpec{register: func(add *HookAdder) {
		add.AfterAppBootstrap(func(ctx context.Context, log *logger.Logger, flag *RunFlag, client *rpcclient.Client) {
			assert.NotNil(t, ctx)
			assert.NotNil(t, log)
			assert.NotNil(t, flag)
			assert.NotNil(t, client)
			assert.Empty(t, a.components)
			assert.Empty(t, a.modules)
			events = append(events, "bootstrap")
		})
		add.AfterAppBootstrap(func() { events = append(events, "bootstrap-second") })

		add.AfterComponentsInitialized(func(c *initHooksComponent) {
			assert.True(t, c.Ready)
			component = c
			events = append(events, "components")
		})
		add.AfterComponentsInitialized(func() { events = append(events, "components-second") })
		add.AfterModulesInitialized(func(m *initHooksModule, c *initHooksComponent, dep *initHooksCommon) {
			assert.Same(t, m.Common, dep)
			assert.True(t, m.Ready)
			assert.Same(t, component, c)
			assert.Same(t, c, m.Component)
			module = m
			events = append(events, "modules")
		})
	}}
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	a = newApp(spec, flags)
	a.Start()
	defer a.StopGracefully()
	assert.Same(t, component, a.components[0])
	assert.Same(t, module, a.modules[0])
	assert.Equal(t, []string{"bootstrap", "bootstrap-second", "components", "components-second", "modules"}, events)
}

func TestInitializationHooksWithoutModulesOrComponents(t *testing.T) {
	events := []string{}
	spec := &initHooksSpec{empty: true, register: func(add *HookAdder) {
		add.AfterAppBootstrap(func() { events = append(events, "bootstrap") })
		add.AfterComponentsInitialized(func(ctx context.Context) { events = append(events, "components") })
		add.AfterModulesInitialized(func(ctx context.Context) { events = append(events, "modules") })
	}}
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	a := newApp(spec, flags)
	a.Start()
	defer a.StopGracefully()
	assert.Equal(t, []string{"bootstrap", "components", "modules"}, events)
}

func TestHookSignatures(t *testing.T) {
	var nilFunc func()
	for _, callback := range []any{nil, 1, nilFunc, func(...string) {}, func() int { return 0 }} {
		assert.Panics(t, func() { new(HookAdder).AfterAppBootstrap(callback) })
		assert.Panics(t, func() { new(HookAdder).AfterComponentsInitialized(callback) })
	}
}

func TestBootstrapRejectsLaterStageDependencies(t *testing.T) {
	for _, callback := range []any{
		func(*initHooksComponent) {},
		func(*initHooksModule) {},
		func(*initHooksCommon) {},
	} {
		flags := _Flags{}
		flags.EnsureRunFlag()
		flags.InitInprocFlag(true)
		a := newApp(&initHooksSpec{}, flags)
		t.Cleanup(a.cancel)
		a.initLinking()
		a.initInjector()
		a.hooks.AfterAppBootstrap(callback)
		assert.Panics(t, func() { a.afterAppBootstrap() })
		assert.Empty(t, a.components)
		assert.Empty(t, a.modules)
	}
}
