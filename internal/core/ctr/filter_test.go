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

	execution := container.NewExecution(reflect.TypeOf(&ctrTestTarget{}), getMethodByName(reflect.TypeOf(&ctrTestTarget{}), "Combine"))
	execution.Execute([]any{3, "alice"})

	assert.Equal(t, []string{
		"trace:before",
		"context:Combine",
		"trace:after",
	}, events)
	assert.Equal(t, []any{"alice:3"}, execution.Results())
}
