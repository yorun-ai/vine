package app

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
)

// Component

type Component interface {
	mustBeComponent()
	ComponentLifecycle

	Bind(b *di.Binder)
}

type BaseComponent struct {
	_BaseComponentLifecycle
}

func (*BaseComponent) mustBeComponent() {}
func (*BaseComponent) Bind(*di.Binder)  {}

func isComponentType(componentType reflect.Type) bool {
	return componentType.Implements(T[Component]())
}

// ManagedComponent delegates initialization and lifecycle to its manager.
type ManagedComponent interface {
	mustBeManagedComponent()
	managerType() reflect.Type
}

// BaseManagedComponent associates a component with its manager type.
type BaseManagedComponent[M ComponentManager] struct{}

func (*BaseManagedComponent[M]) mustBeManagedComponent() {}

func (*BaseManagedComponent[M]) managerType() reflect.Type {
	return T[M]()
}

type ComponentLifecycle interface {
	BeforeAppStart() error
	AfterAppStart()
	BeforeAppStop()
	AfterAppStop()
}

type _BaseComponentLifecycle struct{}

func (*_BaseComponentLifecycle) BeforeAppStart() error { return nil }
func (*_BaseComponentLifecycle) AfterAppStart()        {}
func (*_BaseComponentLifecycle) BeforeAppStop()        {}
func (*_BaseComponentLifecycle) AfterAppStop()         {}

// ComponentManager owns component initialization, bindings, and lifecycle.
type ComponentManager interface {
	ComponentLifecycle
	InitComponent(component ManagedComponent)
	Component() ManagedComponent
	Bind(b *di.Binder)
}

// BaseComponentManager provides default manager methods for embedding.
type BaseComponentManager struct {
	_BaseComponentLifecycle
}

func (*BaseComponentManager) InitComponent(ManagedComponent) {}
func (*BaseComponentManager) Component() ManagedComponent    { return nil }
func (*BaseComponentManager) Bind(*di.Binder)                {}

func resolveComponentManagerTypes(componentTypes []reflect.Type) map[reflect.Type]reflect.Type {
	typeMaps := make(map[reflect.Type]reflect.Type, len(componentTypes))
	for _, componentType := range componentTypes {
		component := reflect.New(componentType.Elem()).Interface().(ManagedComponent)
		typeMaps[componentType] = component.managerType()
	}
	return typeMaps
}
