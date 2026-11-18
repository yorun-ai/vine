package app

import (
	"reflect"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

// _Guard permanently records application spec types and names created in one
// process. Successful creations are never released, including after the
// application stops.
type _Guard struct {
	mutex sync.Mutex
	types map[reflect.Type]struct{}
	names map[string]reflect.Type
}

func newGuard() *_Guard {
	return &_Guard{
		types: map[reflect.Type]struct{}{},
		names: map[string]reflect.Type{},
	}
}

var defaultGuard = newGuard()

func (g *_Guard) create[S ApplicationSpec](enableInproc bool, opts ...FlagApplier) App {
	return g.createByType(T[S](), enableInproc, opts...)
}

func (g *_Guard) createByType(specType reflect.Type, enableInproc bool, opts ...FlagApplier) App {
	g.reserveType(specType)

	name := ""
	succeeded := false
	defer func() {
		if !succeeded {
			g.rollback(specType, name)
		}
	}()

	flags := _Flags{}
	flags.Apply(opts...)
	flags.EnsureRunFlag()
	flags.InitInprocFlag(enableInproc)
	spec := newSpec(specType, flags)
	name = spec.Name()
	g.reserveName(name, specType)
	app := newApp(spec, flags)
	succeeded = true
	return app
}

func (g *_Guard) reserveType(specType reflect.Type) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	vpre.CheckNotOK(g.types, specType, "application %s already created", specType)
	g.types[specType] = struct{}{}
}

func (g *_Guard) reserveName(name string, specType reflect.Type) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	_, exists := g.names[name]
	vpre.Check(!exists, "application name %s already created", name)
	g.names[name] = specType
}

func (g *_Guard) rollback(specType reflect.Type, name string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	delete(g.types, specType)
	if g.names[name] == specType {
		delete(g.names, name)
	}
}
