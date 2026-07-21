package di

import (
	"reflect"
	"testing"

	"go.yorun.ai/vine/util/vslice"

	"github.com/stretchr/testify/assert"
)

type _DependencyCycleA struct {
	Dependency *_DependencyCycleB `inject:""`
}

type _DependencyCycleB struct {
	Dependency *_DependencyCycleC `inject:""`
}

type _DependencyCycleC struct {
	Dependency *_DependencyCycleA `inject:""`
}

type _CaptiveSingleton struct {
	SingletonScoped
	Dependency *_CaptiveTransient `inject:""`
}

type _CaptiveTransient struct {
	TransientScoped
	Dependency *_CaptiveExecution `inject:""`
}

type _CaptiveExecution struct {
	ExecutionScoped
}

type _DependencyFuzz0 struct{}
type _DependencyFuzz1 struct{}
type _DependencyFuzz2 struct{}
type _DependencyFuzz3 struct{}
type _DependencyFuzz4 struct{}
type _DependencyFuzz5 struct{}

var dependencyFuzzTypes = []reflect.Type{
	T[*_DependencyFuzz0](),
	T[*_DependencyFuzz1](),
	T[*_DependencyFuzz2](),
	T[*_DependencyFuzz3](),
	T[*_DependencyFuzz4](),
	T[*_DependencyFuzz5](),
}

func TestDependencyGraphReportsCyclePath(t *testing.T) {
	assert.PanicsWithError(t,
		"cycle dependency detected: *di._DependencyCycleA -> *di._DependencyCycleB -> *di._DependencyCycleC -> *di._DependencyCycleA",
		func() {
			NewInjector(func(b *Binder) {
				b.Bind(T[*_DependencyCycleA]())
			})
		},
	)
}

func TestDependencyGraphRejectsIndirectSingletonToExecutionDependency(t *testing.T) {
	assert.PanicsWithError(t,
		"singleton cannot depend on execution-scoped type: *di._CaptiveSingleton -> *di._CaptiveTransient -> *di._CaptiveExecution",
		func() {
			NewInjector(func(b *Binder) {
				b.Bind(T[*_CaptiveSingleton]())
			})
		},
	)
}

func TestDependencyGraphIgnoresInjectedFieldsOnProvidedInstance(t *testing.T) {
	instance := &_CaptiveSingleton{
		Dependency: &_CaptiveTransient{Dependency: &_CaptiveExecution{}},
	}

	assert.NotPanics(t, func() {
		NewInjector(func(b *Binder) {
			b.BindInstance(instance)
		})
	})
}

func FuzzDependencyGraphScopePaths(f *testing.F) {
	f.Add([]byte{0, 2, 1, 2, 2, 2, 0, 1, 1, 2})
	f.Add([]byte{2, 2, 1, 2, 2, 2, 0, 1, 1, 3, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < len(dependencyFuzzTypes) {
			return
		}

		bounds := make([]*_Bound, len(dependencyFuzzTypes))
		dependencies := make(map[reflect.Type][]reflect.Type, len(dependencyFuzzTypes))
		for index, targetType := range dependencyFuzzTypes {
			binding := newBinding(&_PlainInjector{fallbackScope: TransientScope}, targetType, false)
			binding.explicitScope = fuzzScope(data[index])
			bounds[index] = &_Bound{binding: binding}
		}

		limit := len(data)
		if limit > len(dependencyFuzzTypes)+128 {
			limit = len(dependencyFuzzTypes) + 128
		}
		for index := len(dependencyFuzzTypes); index+1 < limit; index += 2 {
			from := int(data[index]) % len(dependencyFuzzTypes)
			to := int(data[index+1]) % len(dependencyFuzzTypes)
			if from >= to {
				continue
			}
			dependency := dependencyFuzzTypes[to]
			if vslice.Contains(bounds[from].factoryDependencies, dependency) {
				continue
			}
			bounds[from].factoryDependencies = append(bounds[from].factoryDependencies, dependency)
			dependencies[dependencyFuzzTypes[from]] = append(dependencies[dependencyFuzzTypes[from]], dependency)
		}

		graph := newDependencyGraph(bounds)
		if cycle := graph.findCycle(); cycle != nil {
			t.Fatalf("generated DAG reported a cycle: %v", cycle)
		}
		path := graph.findSingletonToExecutionPath()
		expected := dependencyScopePathExists(bounds, dependencies)
		if expected != (path != nil) {
			t.Fatalf("scope reachability mismatch: expected=%t path=%v", expected, path)
		}
		if path != nil {
			if graph.scopeOf(path[0]) != SingletonScope || graph.scopeOf(path[len(path)-1]) != ExecutionScope {
				t.Fatalf("scope path has invalid endpoints: %v", path)
			}
			for index := 0; index < len(path)-1; index++ {
				if !vslice.Contains(dependencies[path[index]], path[index+1]) {
					t.Fatalf("scope path contains missing edge: %v", path)
				}
			}
		}
	})
}

func fuzzScope(value byte) Scope {
	switch value % 3 {
	case 0:
		return SingletonScope
	case 1:
		return ExecutionScope
	default:
		return TransientScope
	}
}

func dependencyScopePathExists(bounds []*_Bound, dependencies map[reflect.Type][]reflect.Type) bool {
	scopes := make(map[reflect.Type]Scope, len(bounds))
	for _, bound := range bounds {
		scopes[bound.TargetType()] = bound.ResolveScope(TransientScope)
	}
	for targetType, scope := range scopes {
		if scope != SingletonScope {
			continue
		}
		queue := []reflect.Type{targetType}
		visited := map[reflect.Type]bool{}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if visited[current] {
				continue
			}
			visited[current] = true
			if current != targetType && scopes[current] == ExecutionScope {
				return true
			}
			queue = append(queue, dependencies[current]...)
		}
	}
	return false
}
