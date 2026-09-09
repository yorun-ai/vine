package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/di"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
)

type lifecycleHooksLog struct {
	FlagModel
	Events *[]string
}

func (l *lifecycleHooksLog) add(event string) { *l.Events = append(*l.Events, event) }

type lifecycleHooksComponent struct {
	BaseComponent
	Log *lifecycleHooksLog `inject:""`
}

func (c *lifecycleHooksComponent) BeforeAppStart() error {
	c.Log.add("component-before-start")
	return nil
}
func (c *lifecycleHooksComponent) AfterAppStart() { c.Log.add("component-after-start") }
func (c *lifecycleHooksComponent) BeforeAppStop() { c.Log.add("component-before-stop") }
func (c *lifecycleHooksComponent) AfterAppStop()  { c.Log.add("component-after-stop") }

type lifecycleHooksModule struct {
	BaseModule
	Log        *lifecycleHooksLog        `inject:""`
	Component  *lifecycleHooksComponent  `inject:""`
	Dependency *lifecycleHooksDependency `inject:""`
}

func (m *lifecycleHooksModule) BeforeAppStart() error { m.Log.add("module-before-start"); return nil }
func (m *lifecycleHooksModule) AfterAppStart()        { m.Log.add("module-after-start") }
func (m *lifecycleHooksModule) BeforeAppStop()        { m.Log.add("module-before-stop") }
func (m *lifecycleHooksModule) AfterAppStop()         { m.Log.add("module-after-stop") }

type lifecycleHooksDependency struct{}

type lifecycleHooksSpec struct {
	Application
	register func(*HookAdder)
	created  *int
	empty    bool
}

func (*lifecycleHooksSpec) Name() string               { return "test.lifecycle.hooks" }
func (s *lifecycleHooksSpec) InitHooks(add *HookAdder) { s.register(add) }
func (s *lifecycleHooksSpec) InitComponents(add TypeAdder) {
	if !s.empty {
		add(T[*lifecycleHooksComponent]())
	}
}
func (s *lifecycleHooksSpec) InitModules(add TypeAdder) {
	if !s.empty {
		add(T[*lifecycleHooksModule]())
	}
}
func (s *lifecycleHooksSpec) BindCommon(b *di.Binder) {
	b.BindFactory(func() *lifecycleHooksDependency {
		*s.created++
		return new(lifecycleHooksDependency)
	}).In(di.SingletonScope)
}

func newLifecycleHooksApp(events *[]string, spec *lifecycleHooksSpec) *_AppImpl {
	flags := _Flags{}
	flags.Apply(With(&lifecycleHooksLog{Events: events}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(true)
	return newApp(spec, flags)
}

func TestApplicationLifecycleHooks(t *testing.T) {
	events := []string{}
	created := 0
	var dependency *lifecycleHooksDependency
	var a *_AppImpl
	spec := &lifecycleHooksSpec{created: &created, register: func(add *HookAdder) {
		add.BeforeAppStart(func(c *lifecycleHooksComponent, m *lifecycleHooksModule, dep *lifecycleHooksDependency, client *rpcclient.Client) error {
			assert.Same(t, a.components[0], c)
			assert.Same(t, a.modules[0], m)
			assert.Same(t, c, m.Component)
			assert.Same(t, m.Dependency, dep)
			assert.NotNil(t, client)
			dependency = dep
			events = append(events, "app-before-start")
			return nil
		})
		add.BeforeAppStart(func() { events = append(events, "app-before-start-second") })
		add.AfterAppStart(func() { events = append(events, "app-after-start") })
		add.BeforeAppStop(func() { events = append(events, "app-before-stop") })
		add.BeforeAppStop(func() { events = append(events, "app-before-stop-second") })
		add.AfterAppStop(func(ctx context.Context, dep *lifecycleHooksDependency) {
			assert.ErrorIs(t, ctx.Err(), context.Canceled)
			assert.Same(t, dependency, dep)
			assert.Equal(t, 1, created)
			events = append(events, "app-after-stop")
		})
		add.AfterAppStop(func() { events = append(events, "app-after-stop-second") })
	}}
	a = newLifecycleHooksApp(&events, spec)
	a.Start()
	a.StopGracefully()
	assert.Equal(t, []string{
		"app-before-start", "app-before-start-second", "component-before-start", "module-before-start",
		"component-after-start", "module-after-start", "app-after-start",
		"app-before-stop-second", "app-before-stop", "module-before-stop", "component-before-stop",
		"module-after-stop", "component-after-stop", "app-after-stop-second", "app-after-stop",
	}, events)
}

func TestBeforeAppStartErrorStopsStartup(t *testing.T) {
	events := []string{}
	created := 0
	spec := &lifecycleHooksSpec{created: &created, register: func(add *HookAdder) {
		add.BeforeAppStart(func() error { return errors.New("hook failed") })
		add.BeforeAppStart(func() { t.Error("later callback ran") })
		add.AfterAppStart(func() { t.Error("after-start callback ran") })
	}}
	a := newLifecycleHooksApp(&events, spec)
	t.Cleanup(a.cancel)
	assert.Panics(t, func() { a.Start() })
	assert.Empty(t, events)
	assert.Nil(t, a.httpServer)
}

func TestLifecycleHookSignatures(t *testing.T) {
	var nilFunc func()
	for _, callback := range []any{nil, 1, nilFunc, func(...string) {}, func() int { return 0 }} {
		hooks := new(HookAdder)
		assert.Panics(t, func() { hooks.BeforeAppStart(callback) })
		assert.Panics(t, func() { hooks.AfterAppStart(callback) })
		assert.Panics(t, func() { hooks.BeforeAppStop(callback) })
		assert.Panics(t, func() { hooks.AfterAppStop(callback) })
	}
	hooks := new(HookAdder)
	assert.NotPanics(t, func() { hooks.BeforeAppStart(func() error { return nil }) })
	assert.Panics(t, func() { hooks.AfterAppStart(func() error { return nil }) })
	assert.Panics(t, func() { hooks.BeforeAppStop(func() error { return nil }) })
	assert.Panics(t, func() { hooks.AfterAppStop(func() error { return nil }) })
}

func TestLifecycleHooksWithoutComponentsOrModules(t *testing.T) {
	events := []string{}
	spec := &lifecycleHooksSpec{empty: true, register: func(add *HookAdder) {
		add.BeforeAppStart(func(ctx context.Context) { assert.NoError(t, ctx.Err()); events = append(events, "before-start") })
		add.AfterAppStart(func() { events = append(events, "after-start") })
		add.BeforeAppStop(func() { events = append(events, "before-stop") })
		add.AfterAppStop(func(ctx context.Context) {
			assert.ErrorIs(t, ctx.Err(), context.Canceled)
			events = append(events, "after-stop")
		})
	}}
	a := newLifecycleHooksApp(&events, spec)
	a.Start()
	a.StopGracefully()
	assert.Empty(t, a.components)
	assert.Empty(t, a.modules)
	assert.Equal(t, []string{"before-start", "after-start", "before-stop", "after-stop"}, events)
}
