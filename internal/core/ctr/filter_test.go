package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/core/di"
)

func TestTargetInvocationFilterInvokesTargetAndStoresResults(t *testing.T) {
	events := []string{}
	container := newCTRTestContainer(
		&events,
		[]reflect.Type{
			reflect.TypeFor[*ctrTraceFilter](),
			reflect.TypeFor[*ctrContextFilter](),
		},
		func(b *di.Binder) {
			b.Bind(di.T[*ctrTestTarget]())
		},
	)

	execution := container.NewExecution(reflect.TypeFor[*ctrTestTarget](), getMethodByName(reflect.TypeFor[*ctrTestTarget](), "Combine"))
	execution.Execute([]any{3, "alice"})

	assert.Equal(t, []string{
		"trace:before",
		"context:Combine",
		"trace:after",
	}, events)
	assert.Equal(t, []any{"alice:3"}, execution.Results())
}

func (*ctrTestTarget) AcceptNil(tag any, names ...*string) bool {
	return tag == nil && len(names) == 2 && names[0] == nil && names[1] == nil
}

func TestTargetInvocationPassesNilToNullableParameters(t *testing.T) {
	kind := di.T[*ctrTestTarget]()
	container := NewContainer(Option{BindAppliers: []di.BindApplier{func(b *di.Binder) { b.Bind(kind) }}})
	execution := container.NewExecution(kind, getMethodByName(kind, "AcceptNil"))
	execution.Execute([]any{nil, nil, (*string)(nil)})
	assert.Equal(t, []any{true}, execution.Results())

	sum := container.NewExecution(kind, getMethodByName(kind, "Sum"))
	assert.PanicsWithError(t, "cannot pass nil to parameter 0 of type int", func() {
		sum.Execute([]any{nil, 1})
	})
}
