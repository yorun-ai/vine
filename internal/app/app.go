package app

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/util/vpre"
)

var (
	appsByType = map[reflect.Type]App{}
	appsByName = map[string]struct{}{}
)

type App interface {
	Name() string
	Start()
	StopGracefully()
	StartAndWait()
}

func New[S ApplicationSpec](opts ...FlagApplier) App {
	return newValidatedByType(T[S](), false, opts...)
}

func NewInproc[S ApplicationSpec](opts ...FlagApplier) App {
	return NewInprocByType(T[S](), opts...)
}

func NewInprocByType(specType reflect.Type, opts ...FlagApplier) App {
	return newValidatedByType(specType, true, opts...)
}

func NewInternal[S interface {
	ApplicationSpec
	InternalApplicationSpec
}](opts ...FlagApplier) App {
	return newInternalByType(T[S](), false, opts...)
}

func NewInternalInproc[S interface {
	ApplicationSpec
	InternalApplicationSpec
}](opts ...FlagApplier) App {
	return newInternalByType(T[S](), true, opts...)
}

func newInternalByType(specType reflect.Type, enableInproc bool, opts ...FlagApplier) App {
	vpre.Check(specType.Implements(T[InternalApplicationSpec]()), "application spec %s is not internal", specType)
	return newValidatedByType(specType, enableInproc, opts...)
}

func newValidatedByType(specType reflect.Type, enableInproc bool, opts ...FlagApplier) App {
	vpre.CheckNotOK(appsByType, specType, "application %s already created", specType)

	flags := _Flags{}
	flags.Apply(opts...)
	flags.EnsureRunFlag()
	flags.InitInprocFlag(enableInproc)

	spec := newSpec(specType, flags)
	_, exists := appsByName[spec.Name()]
	vpre.Check(!exists, "application name %s already created", spec.Name())
	app := newApp(spec, flags)
	appsByType[specType] = app
	appsByName[spec.Name()] = struct{}{}
	return app
}

func newSpec(specType reflect.Type, flags _Flags) ApplicationSpec {
	injector := di.NewInjector(func(b *di.Binder) {
		for flagType, flag := range flags {
			b.Bind(flagType).ToInstance(flag)
		}
		b.Bind(specType).In(di.SingletonScope)
	})
	return injector.Get(specType).Interface().(ApplicationSpec)
}

type TypeAdder func(reflect.Type)

type ApplicationSpec interface {
	Name() string
	InitComponents(addComponent TypeAdder)
	InitModules(addModule TypeAdder)
	BindCommon(b *di.Binder)

	mustBeApplicationSpec()
}

type Application struct {
	AppFlag *RunFlag `inject:""`
}

func (*Application) Name() string {
	return ""
}

func (a *Application) InitComponents(addComponent TypeAdder) {}

func (*Application) InitModules(addModule TypeAdder) {}

func (a *Application) BindCommon(b *di.Binder) {}

func (*Application) mustBeApplicationSpec() {}
