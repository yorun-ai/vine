package app

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/conf"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	taskcore "go.yorun.ai/vine/internal/core/task"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
)

func attachTestLinker(app *_AppImpl) {
	app.linker = &testLinker{
		RpcProxyOutEndpointValue: "http://127.0.0.1:7079/rpc/proxy/out",
	}
}

type testDepsFlag struct {
	FlagModel
	Value string
}

type testComponentBoundDep struct {
	Value string
}

type testModuleBoundDep struct {
	Value string
}

type testSimpleComponentBoundDep struct {
	Value string
}

type testSpecBoundDep struct {
	Value string
}

type testDepsFrameworkComponent struct {
	BaseFrameworkComponent[*testDepsFrameworkComponent]
	BaseFrameworkComponentMinder

	component FrameworkComponent
}

func (*testDepsFrameworkComponent) Bind(b *di.Binder) {
	b.BindInstance(&testComponentBoundDep{Value: "component"})
}

func (c *testDepsFrameworkComponent) InitComponent(component FrameworkComponent) {
	c.component = component
}

func (c *testDepsFrameworkComponent) Component() FrameworkComponent {
	return c.component
}

type testDepsComponent struct {
	testDepsFrameworkComponent
}

type testDepsModule struct {
	BaseModule
}

func (*testDepsModule) Bind(b *di.Binder) {
	b.BindInstance(&testModuleBoundDep{Value: "module"})
}

type testSpecDepsModule struct {
	BaseModule

	Spec *testSpecBoundDep `inject:""`
}

type testSimpleDepsComponent struct {
	BaseComponent
}

func (*testSimpleDepsComponent) Bind(b *di.Binder) {
	b.BindInstance(&testSimpleComponentBoundDep{Value: "simple"})
}

type testDepsConsumer struct {
	Flag   *testDepsFlag                `inject:""`
	Comp   *testComponentBoundDep       `inject:""`
	Simple *testSimpleComponentBoundDep `inject:""`
	Plug   *testModuleBoundDep          `inject:""`
}

type testSingletonDepComponent struct {
	testSingletonDepFrameworkComponent
}

func (c *testSingletonDepComponent) DIInit() {
	*testSingletonComponentInitCount++
}

type testClientComponent struct {
	testClientFrameworkComponent

	Client *rpcclient.Client `inject:""`
}

type testTaskLauncherComponent struct {
	testTaskLauncherFrameworkComponent

	Launcher testComponentTaskLauncher `inject:""`
}

type testEmitterComponent struct {
	testEmitterFrameworkComponent

	Emitter testEventerEmitter `inject:""`
}

type testSingletonConsumerComponent struct {
	testSingletonConsumerFrameworkComponent

	Dependency *testSingletonDepComponent `inject:""`
}

type testSingletonDepFrameworkComponent struct {
	testFrameworkComponent
}

func (*testSingletonDepFrameworkComponent) minderType() reflect.Type {
	return T[*testSingletonDepFrameworkComponent]()
}

type testSingletonConsumerFrameworkComponent struct {
	testFrameworkComponent
}

func (*testSingletonConsumerFrameworkComponent) minderType() reflect.Type {
	return T[*testSingletonConsumerFrameworkComponent]()
}

type testClientFrameworkComponent struct {
	testFrameworkComponent
}

func (*testClientFrameworkComponent) minderType() reflect.Type {
	return T[*testClientFrameworkComponent]()
}

type testTaskLauncherFrameworkComponent struct {
	testFrameworkComponent
}

func (*testTaskLauncherFrameworkComponent) minderType() reflect.Type {
	return T[*testTaskLauncherFrameworkComponent]()
}

type testEmitterFrameworkComponent struct {
	testFrameworkComponent
}

func (*testEmitterFrameworkComponent) minderType() reflect.Type {
	return T[*testEmitterFrameworkComponent]()
}

type testSimpleSingletonDepComponent struct {
	BaseComponent
}

func (c *testSimpleSingletonDepComponent) DIInit() {
	*testSimpleSingletonComponentInitCount++
}

type testSimpleSingletonConsumerComponent struct {
	BaseComponent

	Dependency *testSimpleSingletonDepComponent `inject:""`
}

type testSingletonDepModule struct {
	BaseModule
}

func (p *testSingletonDepModule) DIInit() {
	*testSingletonModuleInitCount++
}

type testSingletonConsumerModule struct {
	BaseModule

	Dependency *testSingletonDepModule `inject:""`
}

type testFlagCopyConsumer struct {
	Flag *testDepsFlag `inject:""`
}

type testConfigValue struct {
	conf.ConfigModel
	Name string `json:"name"`
}

type testConfigConsumer struct {
	Config *testConfigValue `inject:""`
}

type testInternalRuntimeConsumer struct {
	Runtime InternalRuntime `inject:""`
}

type testDepsAppSpec struct {
	Application
	ServicerEnabled
}

func (*testDepsAppSpec) Name() string {
	return "test.deps"
}

func (*testDepsAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testDepsComponent]())
	addComponent(T[*testSimpleDepsComponent]())
}

func (*testDepsAppSpec) InitModules(addModule TypeAdder) {
	addModule(T[*testDepsModule]())
}

func (*testDepsAppSpec) BindCommon(b *di.Binder) {
	b.BindInstance(&testSpecBoundDep{Value: "spec"})
}

type testSpecDepsAppSpec struct {
	Application
	ServicerEnabled
}

func (*testSpecDepsAppSpec) Name() string {
	return "test.spec.deps"
}

type testSingletonComponentAppSpec struct {
	Application
}

func (*testSingletonComponentAppSpec) Name() string {
	return "test.singleton.component"
}

func (*testSingletonComponentAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testSingletonDepComponent]())
	addComponent(T[*testSingletonConsumerComponent]())
}

type testComponentClientAppSpec struct {
	Application
}

func (*testComponentClientAppSpec) Name() string {
	return "test.component.client"
}

func (*testComponentClientAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testClientComponent]())
}

type testComponentTaskLauncherAppSpec struct {
	Application
}

func (*testComponentTaskLauncherAppSpec) Name() string {
	return "test.component.tasklauncher"
}

func (*testComponentTaskLauncherAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testTaskLauncherComponent]())
}

type testComponentEmitterAppSpec struct {
	Application
}

func (*testComponentEmitterAppSpec) Name() string {
	return "test.component.emitter"
}

func (*testComponentEmitterAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testEmitterComponent]())
}

type testUserComponentBindingAppSpec struct {
	Application
}

func (*testUserComponentBindingAppSpec) Name() string {
	return "test.component.userbinding"
}

func (*testUserComponentBindingAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testSingletonDepComponent]())
	addComponent(T[*testSingletonConsumerComponent]())
}

type testSingletonModuleAppSpec struct {
	Application
}

func (*testSingletonModuleAppSpec) Name() string {
	return "test.singleton.module"
}

func (*testSingletonModuleAppSpec) InitModules(addModule TypeAdder) {
	addModule(T[*testSingletonDepModule]())
	addModule(T[*testSingletonConsumerModule]())
}

type testSimpleComponentAppSpec struct {
	Application
}

func (*testSimpleComponentAppSpec) Name() string {
	return "test.simple.component"
}

func (*testSimpleComponentAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testSimpleSingletonDepComponent]())
	addComponent(T[*testSimpleSingletonConsumerComponent]())
}

var (
	testSingletonComponentInitCount       = new(int)
	testSimpleSingletonComponentInitCount = new(int)
	testSingletonModuleInitCount          = new(int)
	testConfigRegisterOnce                sync.Once
	testComponentTaskRegisterOnce         sync.Once
)

type testComponentTaskLauncher interface {
	mustBeTestComponentTaskLauncher()
}

type defaultTestComponentTaskLauncher struct{}

func (*defaultTestComponentTaskLauncher) mustBeTestComponentTaskLauncher() {}

type testComponentTaskRunner interface {
	mustBeTestComponentTaskRunner()
}

type defaultTestComponentTaskRunner struct{}

func (*defaultTestComponentTaskRunner) mustBeTestComponentTaskRunner() {}

type testComponentTaskRunnerER interface {
	mustBeTestComponentTaskRunnerER()
}

type _WrapperTestComponentTaskRunnerER struct{}

func newWrapperTestComponentTaskRunnerER(testComponentTaskRunner) testComponentTaskRunnerER {
	return &_WrapperTestComponentTaskRunnerER{}
}

func (*_WrapperTestComponentTaskRunnerER) mustBeTestComponentTaskRunnerER() {}

type defaultTestComponentTaskRunnerER struct {
	_WrapperTestComponentTaskRunnerER
}

func ensureTestComponentTaskRegistered() {
	testComponentTaskRegisterOnce.Do(func() {
		taskspec.Register(&taskspec.TaskSpec{
			Name:                "TestComponentTask",
			SkelName:            "test.component.TestComponentTask",
			RunnerType:          reflect.TypeFor[testComponentTaskRunner](),
			DefaultRunnerType:   reflect.TypeFor[*defaultTestComponentTaskRunner](),
			ERRunnerType:        reflect.TypeFor[testComponentTaskRunnerER](),
			WrapperERRunnerCtor: newWrapperTestComponentTaskRunnerER,
			DefaultERRunnerType: reflect.TypeFor[*defaultTestComponentTaskRunnerER](),
			LauncherType:        reflect.TypeFor[testComponentTaskLauncher](),
			LauncherCtor:        func(*taskcore.Launcher) testComponentTaskLauncher { return &defaultTestComponentTaskLauncher{} },
			Triggers: []*taskspec.TriggerSpec{{
				Name:     "Full",
				SkelName: "full",
			}},
		})
	})
}

func registerTestConfigValue() string {
	const skelName = "app.test.Config"
	testConfigRegisterOnce.Do(func() {
		conf.Register(conf.ConfigSpec{
			Name:      "TestConfigValue",
			SkelName:  skelName,
			Lifecycle: conf.LifecycleEternal,
			Type:      reflect.TypeFor[*testConfigValue](),
		})
	})
	return skelName
}

func (*testSpecDepsAppSpec) InitModules(addModule TypeAdder) {
	addModule(T[*testSpecDepsModule]())
}

func (*testSpecDepsAppSpec) BindCommon(b *di.Binder) {
	b.BindInstance(&testSpecBoundDep{Value: "spec"})
}

func TestAppDepsProvidesFlagComponentAndModuleBindings(t *testing.T) {
	spec := &testDepsAppSpec{AppFlag: &RunFlag{}}
	flags := _Flags{}
	flags.Apply(With(&testDepsFlag{Value: "flag"}))
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(spec, flags)
	app.initInjector()
	app.initComponents()
	app.initModules()

	injector := di.NewInjector(
		app.bindAppDeps,
		func(b *di.Binder) {
			b.Bind(T[meta.Context]()).ToInstance(newMetaContext(context.Background()))
			b.Bind(T[*testDepsConsumer]()).In(di.TransientScope)
		},
	)
	consumer := injector.Get(T[*testDepsConsumer]()).Interface().(*testDepsConsumer)

	assert.Equal(t, "flag", consumer.Flag.Value)
	assert.Equal(t, "component", consumer.Comp.Value)
	assert.Equal(t, "simple", consumer.Simple.Value)
	assert.Equal(t, "module", consumer.Plug.Value)
}

func TestBindFlagsProvidesCopiedFlagInstances(t *testing.T) {
	original := &testDepsFlag{Value: "flag"}
	app := newTestAppImpl()
	app.flags = _Flags{
		T[*testDepsFlag](): original,
	}

	injector := di.NewInjector(
		app.bindRuntime,
		func(b *di.Binder) {
			b.Bind(T[*testFlagCopyConsumer]()).In(di.TransientScope)
		},
	)

	consumer := injector.Get(T[*testFlagCopyConsumer]()).Interface().(*testFlagCopyConsumer)
	consumer.Flag.Value = "changed"

	assert.Equal(t, "flag", original.Value)
	assert.NotSame(t, original, consumer.Flag)
}

func TestBindRuntimeProvidesConfigViaLinker(t *testing.T) {
	skelName := registerTestConfigValue()
	app := newTestAppImpl()
	app.linker = &testLinker{
		EternalConfigByKey: map[string]string{
			skelName: `{"name":"from-link"}`,
		},
	}
	app.reader = conf.NewReader(app.linker)

	injector := di.NewInjector(
		app.bindRuntime,
		func(b *di.Binder) {
			b.Bind(T[*testConfigConsumer]()).In(di.TransientScope)
		},
	)

	consumer := injector.Get(T[*testConfigConsumer]()).Interface().(*testConfigConsumer)
	assert.NotNil(t, consumer.Config)
	assert.Equal(t, "from-link", consumer.Config.Name)
}

func TestBindRuntimeProvidesInternalRuntimeOnlyForInternalApplications(t *testing.T) {
	bindConsumer := func(b *di.Binder) {
		b.Bind(T[*testInternalRuntimeConsumer]()).In(di.TransientScope)
	}

	ordinaryApp := newTestAppImpl()
	assert.Panics(t, func() {
		di.NewInjector(ordinaryApp.bindRuntime, bindConsumer)
	})

	internalApp := newTestAppImpl()
	internalApp.spec = &testInternalOnlyAppSpec{}
	internalInjector := di.NewInjector(internalApp.bindRuntime, bindConsumer)
	consumer := internalInjector.Get(T[*testInternalRuntimeConsumer]()).Interface().(*testInternalRuntimeConsumer)

	assert.Same(t, internalApp, consumer.Runtime)
}

func TestInitModulesProvidesDomainBindings(t *testing.T) {
	spec := &testSpecDepsAppSpec{AppFlag: &RunFlag{}}
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(spec, flags)
	app.initInjector()

	app.initModules()

	module, ok := app.modules[0].(*testSpecDepsModule)
	assert.True(t, ok)
	assert.NotNil(t, module.Spec)
	assert.Equal(t, "spec", module.Spec.Value)
}

func TestInitComponentsBindsComponentTypesAsSingletons(t *testing.T) {
	*testSingletonComponentInitCount = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testSingletonComponentAppSpec{AppFlag: &RunFlag{}}, flags)
	app.initInjector()

	app.initComponents()

	dependency, ok := app.frameworkComponentMinders[0].Component().(*testSingletonDepComponent)
	if !assert.True(t, ok) {
		return
	}
	consumer, ok := app.frameworkComponentMinders[1].Component().(*testSingletonConsumerComponent)
	if !assert.True(t, ok) {
		return
	}

	assert.Same(t, dependency, consumer.Dependency)
	assert.Equal(t, 1, *testSingletonComponentInitCount)
}

func TestInitComponentsBindsSimpleComponentTypesAsSingletons(t *testing.T) {
	*testSimpleSingletonComponentInitCount = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testSimpleComponentAppSpec{AppFlag: &RunFlag{}}, flags)
	app.initInjector()

	app.initComponents()

	dependency, ok := app.components[0].(*testSimpleSingletonDepComponent)
	if !assert.True(t, ok) {
		return
	}
	consumer, ok := app.components[1].(*testSimpleSingletonConsumerComponent)
	if !assert.True(t, ok) {
		return
	}

	assert.Same(t, dependency, consumer.Dependency)
	assert.Equal(t, 1, *testSimpleSingletonComponentInitCount)
}

func TestInitComponentsProvidesRpcClient(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testComponentClientAppSpec{AppFlag: &RunFlag{}}, flags)
	attachTestLinker(app)
	app.initInjector()

	app.initComponents()

	component, ok := app.frameworkComponentMinders[0].Component().(*testClientComponent)
	if !assert.True(t, ok) {
		return
	}

	assert.NotNil(t, component.Client)
}

func TestInitComponentsProvidesTaskLauncher(t *testing.T) {
	ensureTestComponentTaskRegistered()

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testComponentTaskLauncherAppSpec{AppFlag: &RunFlag{}}, flags)
	attachTestLinker(app)
	app.initInjector()

	app.initComponents()

	component, ok := app.frameworkComponentMinders[0].Component().(*testTaskLauncherComponent)
	if !assert.True(t, ok) {
		return
	}

	assert.NotNil(t, component.Launcher)
}

func TestInitComponentsProvidesEmitter(t *testing.T) {
	ensureEventerEventRegistered()

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testComponentEmitterAppSpec{AppFlag: &RunFlag{}}, flags)
	attachTestLinker(app)
	app.initInjector()

	app.initComponents()

	component, ok := app.frameworkComponentMinders[0].Component().(*testEmitterComponent)
	if !assert.True(t, ok) {
		return
	}

	assert.NotNil(t, component.Emitter)
}

func TestBindComponentsExposesUserComponentType(t *testing.T) {
	*testSingletonComponentInitCount = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testUserComponentBindingAppSpec{AppFlag: &RunFlag{}}, flags)
	attachTestLinker(app)
	app.initInjector()
	app.initComponents()

	injector := app.injector.SubInjector(app.bindClients, app.bindComponents)
	dependency := injector.Get(T[*testSingletonDepComponent]()).Interface().(*testSingletonDepComponent)
	consumer := injector.Get(T[*testSingletonConsumerComponent]()).Interface().(*testSingletonConsumerComponent)

	assert.Same(t, dependency, consumer.Dependency)
	assert.Equal(t, 1, *testSingletonComponentInitCount)
}

func TestBindComponentsExposesSimpleComponentBindings(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testDepsAppSpec{AppFlag: &RunFlag{}}, flags)
	attachTestLinker(app)
	app.initInjector()
	app.initComponents()

	injector := app.injector.SubInjector(app.bindClients, app.bindComponents)
	simpleDep := injector.Get(T[*testSimpleComponentBoundDep]()).Interface().(*testSimpleComponentBoundDep)

	assert.Equal(t, "simple", simpleDep.Value)
}

func TestInitComponentsBindsFrameworkComponentMinderTypes(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testUserComponentBindingAppSpec{AppFlag: &RunFlag{}}, flags)
	app.initInjector()

	app.initComponents()

	if len(app.frameworkComponentMinders) != 2 {
		t.Fatalf("expected 2 component minders, got %d", len(app.frameworkComponentMinders))
	}
	if _, ok := app.frameworkComponentMinders[0].(*testSingletonDepFrameworkComponent); !ok {
		t.Fatalf("expected first component minder type *testSingletonDepFrameworkComponent, got %T", app.frameworkComponentMinders[0])
	}
	if _, ok := app.frameworkComponentMinders[1].(*testSingletonConsumerFrameworkComponent); !ok {
		t.Fatalf("expected second component minder type *testSingletonConsumerFrameworkComponent, got %T", app.frameworkComponentMinders[1])
	}
}

func TestInitModulesBindsModuleTypesAsSingletons(t *testing.T) {
	*testSingletonModuleInitCount = 0

	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testSingletonModuleAppSpec{AppFlag: &RunFlag{}}, flags)
	app.initInjector()

	app.initModules()

	dependency, ok := app.modules[0].(*testSingletonDepModule)
	if !assert.True(t, ok) {
		return
	}
	consumer, ok := app.modules[1].(*testSingletonConsumerModule)
	if !assert.True(t, ok) {
		return
	}

	assert.Same(t, dependency, consumer.Dependency)
	assert.Equal(t, 1, *testSingletonModuleInitCount)
}

type managedReadyLog struct {
	events          []string
	initializations int
}

type ManagedReadyResource struct {
	BaseFrameworkComponent[*managedReadyMinder]
	Log        *managedReadyLog `inject:""`
	configured bool
	ready      bool
}

func (r *ManagedReadyResource) DIInit() { r.configured = true }

type managedReadyOtherResource struct{ ManagedReadyResource }

type managedReadyMinder struct {
	BaseFrameworkComponentMinder
	component FrameworkComponent
}

func (m *managedReadyMinder) InitComponent(component FrameworkComponent) {
	m.component = component
	var resource *ManagedReadyResource
	switch c := component.(type) {
	case *ManagedReadyResource:
		resource = c
	case *managedReadyOtherResource:
		resource = &c.ManagedReadyResource
	}
	if !resource.configured {
		panic("component DIInit must run before minder")
	}
	resource.ready = true
	resource.Log.initializations++
}
func (m *managedReadyMinder) Component() FrameworkComponent { return m.component }
func (m *managedReadyMinder) BeforeAppStart() error {
	switch c := m.component.(type) {
	case *ManagedReadyResource:
		c.Log.events = append(c.Log.events, "resource.start")
	case *managedReadyOtherResource:
		c.Log.events = append(c.Log.events, "other.start")
	}
	return nil
}
func (m *managedReadyMinder) AfterAppStop() {
	switch c := m.component.(type) {
	case *ManagedReadyResource:
		c.Log.events = append(c.Log.events, "resource.stop")
	case *managedReadyOtherResource:
		c.Log.events = append(c.Log.events, "other.stop")
	}
}

type managedReadyConsumer struct {
	BaseComponent
	Resource *ManagedReadyResource      `inject:""`
	Other    *managedReadyOtherResource `inject:""`
}

func (c *managedReadyConsumer) DIInit() {
	if !c.Resource.ready || !c.Other.ready {
		panic("resource minder has not initialized")
	}
}
func (c *managedReadyConsumer) BeforeAppStart() error {
	c.Resource.Log.events = append(c.Resource.Log.events, "consumer.start")
	return nil
}
func (c *managedReadyConsumer) AfterAppStop() {
	c.Resource.Log.events = append(c.Resource.Log.events, "consumer.stop")
}

type managedReadyAppSpec struct {
	Application
	componentTypes []reflect.Type
}

func (*managedReadyAppSpec) Name() string { return "test.managedready" }
func (s *managedReadyAppSpec) InitComponents(add TypeAdder) {
	for _, kind := range s.componentTypes {
		add(kind)
	}
}

func TestInitComponentsReadiesManagedDependenciesBeforeConsumer(t *testing.T) {
	log := new(managedReadyLog)
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	a := newApp(&managedReadyAppSpec{componentTypes: []reflect.Type{
		T[*managedReadyConsumer](), T[*managedReadyOtherResource](), T[*ManagedReadyResource](),
	}}, flags)
	a.initInjector()
	a.injector = a.injector.SubInjector(func(b *di.Binder) { b.BindInstance(log) })
	a.initComponents()
	assert.Equal(t, 2, log.initializations)
	consumer := a.components[0].(*managedReadyConsumer)
	assert.Same(t, consumer.Other, a.frameworkComponentMinders[0].Component())
	assert.Same(t, consumer.Resource, a.frameworkComponentMinders[1].Component())
	assert.NotSame(t, a.frameworkComponentMinders[0], a.frameworkComponentMinders[1])
	child := a.injector.SubInjector(a.bindComponents)
	assert.Same(t, consumer.Resource, child.Get(T[*ManagedReadyResource]()).Interface())
	assert.Equal(t, 2, log.initializations)
	assert.NoError(t, a.beforeAppStart())
	a.afterAppStop()
	assert.Equal(t, []string{"consumer.start", "other.start", "resource.start", "resource.stop", "other.stop", "consumer.stop"}, log.events)
}

type managedReadyViaMinder struct {
	BaseFrameworkComponent[*managedReadyDependentMinder]
	ready bool
}

type managedReadyDependentMinder struct {
	BaseFrameworkComponentMinder
	Resource  *ManagedReadyResource `inject:""`
	component *managedReadyViaMinder
}

func (m *managedReadyDependentMinder) InitComponent(component FrameworkComponent) {
	if !m.Resource.ready {
		panic("minder dependency is not ready")
	}
	m.component = component.(*managedReadyViaMinder)
	m.component.ready = true
}
func (m *managedReadyDependentMinder) Component() FrameworkComponent { return m.component }

type managedReadyCycle struct {
	BaseFrameworkComponent[*managedReadyCycleMinder]
}
type managedReadyCycleMinder struct {
	BaseFrameworkComponentMinder
	Consumer *managedReadyCycleConsumer `inject:""`
}
type managedReadyCycleConsumer struct {
	BaseComponent
	Resource *managedReadyCycle `inject:""`
}

func TestInitComponentsReadiesMinderDependencies(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	a := newApp(&managedReadyAppSpec{componentTypes: []reflect.Type{
		T[*managedReadyViaMinder](), T[*ManagedReadyResource](),
	}}, flags)
	a.initInjector()
	a.injector = a.injector.SubInjector(func(b *di.Binder) { b.BindInstance(new(managedReadyLog)) })
	a.initComponents()
	minder := a.frameworkComponentMinders[0].(*managedReadyDependentMinder)
	assert.True(t, minder.component.ready)
	assert.True(t, minder.Resource.ready)
	assert.Same(t, minder.Resource, a.frameworkComponentMinders[1].Component())
}

func TestInitComponentsRejectsCyclesThroughMinder(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	a := newApp(&managedReadyAppSpec{componentTypes: []reflect.Type{
		T[*managedReadyCycleConsumer](), T[*managedReadyCycle](),
	}}, flags)
	a.initInjector()
	assert.PanicsWithError(t, "cycle dependency detected: *app.managedReadyCycle -> *app.managedReadyCycleMinder -> *app.managedReadyCycleConsumer -> *app.managedReadyCycle", a.initComponents)
}
