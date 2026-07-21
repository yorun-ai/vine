package di

import (
	"reflect"

	"go.yorun.ai/vine/util/vmath"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type _DependencyGraph struct {
	bounds      map[reflect.Type]*_Bound
	graph       *vmath.Graph[reflect.Type]
	targetTypes []reflect.Type
}

func newDependencyGraph(bounds []*_Bound) *_DependencyGraph {
	sortedBounds := vslice.SortBy(bounds, func(left, right *_Bound) bool {
		return compareTypeName(left.TargetType(), right.TargetType()) < 0
	})
	graph := &_DependencyGraph{
		bounds:      make(map[reflect.Type]*_Bound, len(bounds)),
		graph:       vmath.NewGraph[reflect.Type](),
		targetTypes: make([]reflect.Type, 0, len(bounds)),
	}
	for _, bound := range sortedBounds {
		targetType := bound.TargetType()
		graph.bounds[targetType] = bound
		graph.graph.AddNode(targetType)
		graph.graph.AddEdge(targetType, bound.Dependencies()...)
		graph.targetTypes = append(graph.targetTypes, targetType)
	}
	return graph
}

func compareTypeName(left, right reflect.Type) int {
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
	graph := newDependencyGraph(i.visibleBounds())
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
