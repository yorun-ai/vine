package di

import (
	"reflect"

	"go.yorun.ai/vine/util/vpre"
)

type SeedApplier func(s *Seeder)

type Seeder struct {
	injector *_ExecutionInjector
}

func newSeeder(injector *_ExecutionInjector) *Seeder {
	return &Seeder{
		injector: injector,
	}
}

func (s *Seeder) seed(targetType reflect.Type, instance reflect.Value) {
	done := s.injector.beginSeed()
	defer done()

	vpre.Check(s.injector.canGet(targetType), "type %s is not bind", targetType)

	instanceType := instance.Type()
	vpre.Check(instanceType.AssignableTo(targetType), "instance type %s is not compatible with %s", instanceType, targetType)

	resolver, ok := s.injector.GetResolver(targetType).(*_PersistedResolver)
	vpre.Check(ok && resolver.bound.ResolveScope(s.injector.fallbackScope) == ExecutionScope, "type %s is not execution scoped, cannot be seeded", targetType)
	resolver.SeedInstance(instance)
}

func (s *Seeder) Seed(targetType reflect.Type, instance any) {
	s.seed(targetType, reflect.ValueOf(instance))
}

func (s *Seeder) SeedInstance(instance any) {
	s.seed(reflect.TypeOf(instance), reflect.ValueOf(instance))
}
