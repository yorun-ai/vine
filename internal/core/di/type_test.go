package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type typeTestStruct struct{}

type typeTestInterface interface {
	Run()
}

func TestCheckBindType(t *testing.T) {
	assert.NotPanics(t, func() { checkBindType(T[typeTestInterface]()) })
	assert.NotPanics(t, func() { checkBindType(T[*typeTestStruct]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[map[string]int]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[[]string]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[func()]()) })

	assert.Panics(t, func() { checkBindType(reflect.TypeFor[typeTestStruct]()) })
}

func TestParseResolveTargetTypePanics(t *testing.T) {
	assert.Panics(t, func() {
		parseResolveTargetType(typeTestStruct{})
	})

	value := typeTestStruct{}
	assert.Panics(t, func() {
		parseResolveTargetType(&value)
	})
}
