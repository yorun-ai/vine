package app

import "testing"

type _GuardTestSpec struct {
	Application
}

func (*_GuardTestSpec) Name() string {
	return "test.guard"
}

func TestGuardInstancesAreIsolated(t *testing.T) {
	first := newGuard()
	second := newGuard()
	first.create[*_GuardTestSpec](false)
	second.create[*_GuardTestSpec](false)
}

func TestGuardRollsBackFailedCreation(t *testing.T) {
	guard := newGuard()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected app creation panic")
			}
		}()
		guard.create[*_GuardTestSpec](false, func(_Flags) {
			panic("creation failed")
		})
	}()

	guard.create[*_GuardTestSpec](false)
}

func TestGuardAllowsOnlyOneConcurrentCreation(t *testing.T) {
	guard := newGuard()
	results := make(chan bool, 32)

	for range cap(results) {
		go func() {
			created := false
			defer func() {
				_ = recover()
				results <- created
			}()
			guard.create[*_GuardTestSpec](false)
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
