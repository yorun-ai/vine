package app

import (
	"reflect"
	"testing"
)

type _GuardTestSpec struct {
	Application
	name string
}

func (s *_GuardTestSpec) Name() string {
	return s.name
}

func TestGuardInstancesAreIsolated(t *testing.T) {
	specType := reflect.TypeFor[*_GuardTestSpec]()
	createSpec := func() ApplicationSpec { return &_GuardTestSpec{name: "test.guard"} }
	createApp := func(ApplicationSpec) App { return &stubApp{} }

	first := newGuard()
	second := newGuard()
	first.create(specType, createSpec, createApp)
	second.create(specType, createSpec, createApp)
}

func TestGuardRollsBackFailedCreation(t *testing.T) {
	guard := newGuard()
	specType := reflect.TypeFor[*_GuardTestSpec]()
	createSpec := func() ApplicationSpec { return &_GuardTestSpec{name: "test.guard.rollback"} }

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected app creation panic")
			}
		}()
		guard.create(specType, createSpec, func(ApplicationSpec) App {
			panic("creation failed")
		})
	}()

	guard.create(specType, createSpec, func(ApplicationSpec) App { return &stubApp{} })
}

func TestGuardAllowsOnlyOneConcurrentCreation(t *testing.T) {
	guard := newGuard()
	specType := reflect.TypeFor[*_GuardTestSpec]()
	results := make(chan bool, 32)

	for range cap(results) {
		go func() {
			created := false
			defer func() {
				_ = recover()
				results <- created
			}()
			guard.create(
				specType,
				func() ApplicationSpec { return &_GuardTestSpec{name: "test.guard.concurrent"} },
				func(ApplicationSpec) App { return &stubApp{} },
			)
			created = true
		}()
	}

	createdCount := 0
	for range cap(results) {
		if <-results {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created app count = %d, want 1", createdCount)
	}
}
