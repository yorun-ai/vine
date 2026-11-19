package di

import (
	"maps"
	"reflect"
	"slices"

	"go.yorun.ai/vine/util/vmath"
	"go.yorun.ai/vine/util/vpre"
)

type _DependencyGraph struct {
	bounds      map[reflect.Type]*_Bound
	graph       *vmath.Graph[reflect.Type]
	targetTypes []reflect.Type
}

func newDependencyGraph(injector *_PlainInjector) *_DependencyGraph {
	graph := &_DependencyGraph{
		bounds:      injector.bounds,
		graph:       vmath.NewGraph[reflect.Type](),
		targetTypes: slices.SortedFunc(maps.Keys(injector.bounds), compareTypeName),
	}
	for _, kind := range graph.targetTypes {
		bound := injector.bounds[kind]
		graph.graph.AddNode(kind)
		// Inherited singletons resolve dependencies in their owning injector,
		// whose graph was already checked.
		if bound.binding.injector != injector && graph.scopeOf(kind) == SingletonScope {
			continue
		}
		graph.graph.AddEdge(kind, bound.Dependencies()...)
	}
	return graph
}

func compareTypeName(left reflect.Type, right reflect.Type) int {
	switch {
	case left.String() < right.String():
		return -1
	case left.String() > right.String():
		return 1
	default:
		return 0
	}
}

func (i *_PlainInjector) checkDependencies() {
	graph := newDependencyGraph(i)
	if cycle := graph.findCycle(); cycle != nil {
		vpre.Panicf("cycle dependency detected: %s", _BuildStack(cycle))
	}
	if path := graph.findSingletonToExecutionPath(); path != nil {
		vpre.Panicf("singleton cannot depend on execution-scoped type: %s", _BuildStack(path))
	}
}

func (g *_DependencyGraph) findCycle() []reflect.Type {
	return g.graph.FindCyclePath()
}

func (g *_DependencyGraph) findSingletonToExecutionPath() []reflect.Type {
	for _, targetType := range g.targetTypes {
		if g.scopeOf(targetType) != SingletonScope {
			continue
		}
		path := g.graph.FindPath(targetType, func(dependency reflect.Type) bool {
			return g.scopeOf(dependency) == ExecutionScope
		})
		if path != nil {
			return path
		}
	}
	return nil
}

func (g *_DependencyGraph) scopeOf(targetType reflect.Type) Scope {
	bound := g.bounds[targetType]
	if bound == nil {
		return noScope
	}
	return bound.ResolveScope(bound.binding.injector.fallbackScope)
}
