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

// FrameworkComponent

type FrameworkComponent interface {
	mustBeFrameworkComponent()
	minderType() reflect.Type
}

type BaseFrameworkComponent[M FrameworkComponentMinder] struct{}

func (*BaseFrameworkComponent[M]) mustBeFrameworkComponent() {}

func (*BaseFrameworkComponent[M]) minderType() reflect.Type {
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

type FrameworkComponentMinder interface {
	ComponentLifecycle
	InitComponent(component FrameworkComponent)
	Component() FrameworkComponent
	Bind(b *di.Binder)
}

type BaseFrameworkComponentMinder struct {
	_BaseComponentLifecycle
}

func (*BaseFrameworkComponentMinder) InitComponent(FrameworkComponent) {}
func (*BaseFrameworkComponentMinder) Component() FrameworkComponent    { return nil }
func (*BaseFrameworkComponentMinder) Bind(*di.Binder)                  {}

func resolveFrameworkComponentMinderTypes(componentTypes []reflect.Type) map[reflect.Type]reflect.Type {
	typeMaps := make(map[reflect.Type]reflect.Type, len(componentTypes))
	for _, componentType := range componentTypes {
		component := reflect.New(componentType.Elem()).Interface().(FrameworkComponent)
		typeMaps[componentType] = component.minderType()
	}
	return typeMaps
}
