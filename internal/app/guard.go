package app

import (
	"reflect"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

type _GuardEntry struct{}

// _Guard permanently records application spec types and names created in one
// process. Successful creations are never released, including after the
// application stops.
type _Guard struct {
	mutex sync.Mutex
	types map[reflect.Type]*_GuardEntry
	names map[string]*_GuardEntry
}

func newGuard() *_Guard {
	return &_Guard{
		types: map[reflect.Type]*_GuardEntry{},
		names: map[string]*_GuardEntry{},
	}
}

var defaultGuard = newGuard()

func (g *_Guard) create[S ApplicationSpec](enableInproc bool, opts ...FlagApplier) App {
	return g.createByType(T[S](), enableInproc, opts...)
}

func (g *_Guard) createByType(specType reflect.Type, enableInproc bool, opts ...FlagApplier) App {
	entry := new(_GuardEntry)
	g.reserveType(specType, entry)

	name := ""
	succeeded := false
	defer func() {
		if !succeeded {
			g.rollback(specType, name, entry)
		}
	}()

	flags := _Flags{}
	flags.Apply(opts...)
	flags.EnsureRunFlag()
	flags.InitInprocFlag(enableInproc)
	spec := newSpec(specType, flags)
	name = spec.Name()
	g.reserveName(name, entry)
	app := newApp(spec, flags)
	succeeded = true
	return app
}

func (g *_Guard) reserveType(specType reflect.Type, entry *_GuardEntry) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	vpre.CheckNotOK(g.types, specType, "application %s already created", specType)
	g.types[specType] = entry
}

func (g *_Guard) reserveName(name string, entry *_GuardEntry) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	_, exists := g.names[name]
	vpre.Check(!exists, "application name %s already created", name)
	g.names[name] = entry
}

func (g *_Guard) rollback(specType reflect.Type, name string, entry *_GuardEntry) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.types[specType] == entry {
		delete(g.types, specType)
	}
	if g.names[name] == entry {
		delete(g.names, name)
	}
}
