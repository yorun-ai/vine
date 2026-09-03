package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/core/di"
)

func TestExecutionSupportsAdditionalSeedAppliers(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(
		&events,
		[]reflect.Type{reflect.TypeFor[*ctrSeededFilter]()},
		func(b *di.Binder) {
			b.Bind(di.T[*ctrTestTarget]())
			b.Bind(di.T[*ctrSeededMessage]())
		},
	)

	execution := container.NewExecution(reflect.TypeFor[*ctrTestTarget](), getMethodByName(reflect.TypeFor[*ctrTestTarget](), "Sum"))
	execution.Execute([]any{2, 7}, func(s *di.Seeder) {
		s.SeedInstance(&ctrSeededMessage{Value: "hello"})
	})

	assert.Equal(t, []string{"seed:hello"}, events)
	assert.Equal(t, []any{9}, execution.Results())
}

func TestExecutionReleasesExecutionScopedFiltersOnPanic(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(
		&events,
		[]reflect.Type{reflect.TypeFor[*ctrDisposeFilter]()},
		func(b *di.Binder) {
			b.Bind(di.T[*ctrPanicTarget]())
		},
	)

	execution := container.NewExecution(reflect.TypeFor[*ctrPanicTarget](), getMethodByName(reflect.TypeFor[*ctrPanicTarget](), "Boom"))

	assert.PanicsWithValue(t, "boom", func() {
		execution.Execute(nil)
	})

	assert.Equal(t, []string{"dispose:before", "dispose:filter"}, events)
}

func TestNewExecutionPanicsWhenTargetMethodIsZeroValue(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(&events, nil, func(b *di.Binder) {
		b.Bind(di.T[*ctrTestTarget]())
	})

	assert.Panics(t, func() {
		_ = container.NewExecution(reflect.TypeFor[*ctrTestTarget](), reflect.Method{})
	})
}
