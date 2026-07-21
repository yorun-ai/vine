package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextArgumentsReturnsClone(t *testing.T) {
	context := newContext(reflect.TypeOf(&ctrTestTarget{}), "Sum")
	context.SetArguments([]any{1, "alice"})

	arguments := context.Arguments()
	arguments[0] = 99

	assert.Equal(t, []any{1, "alice"}, context.Arguments())
}

func TestContextRejectsMutationAfterExecution(t *testing.T) {
	context := newContext(reflect.TypeOf(&ctrTestTarget{}), "Sum")
	context.markFinished()

	assert.Panics(t, func() {
		context.SetTargetType(reflect.TypeOf(&ctrPanicTarget{}))
	})
	assert.Panics(t, func() {
		context.SetTargetMethodName("Other")
	})
	assert.Panics(t, func() {
		context.SetArguments([]any{1})
	})
}

func TestContextStoresResults(t *testing.T) {
	context := newContext(reflect.TypeOf(&ctrTestTarget{}), "Sum")
	results := []any{3}
	context.SetResults(results)
	results[0] = 99

	assert.Equal(t, []any{3}, context.Results())
}

func TestContextResultsReturnsClone(t *testing.T) {
	context := newContext(reflect.TypeOf(&ctrTestTarget{}), "Sum")
	context.SetResults([]any{3, "alice"})

	results := context.Results()
	results[0] = 99

	assert.Equal(t, []any{3, "alice"}, context.Results())
}
