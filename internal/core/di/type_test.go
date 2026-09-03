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

func TestAnalyzeBindType(t *testing.T) {
	assert.Equal(t, bindTypeInterface, analyzeBindType(T[typeTestInterface]()))
	assert.Equal(t, bindTypeStructPointer, analyzeBindType(T[*typeTestStruct]()))
	assert.Equal(t, bindTypeMap, analyzeBindType(reflect.TypeFor[map[string]int]()))
	assert.Equal(t, bindTypeSlice, analyzeBindType(reflect.TypeFor[[]string]()))
	assert.Equal(t, bindTypeFunc, analyzeBindType(reflect.TypeFor[func()]()))
	assert.Equal(t, bindTypeNone, analyzeBindType(reflect.TypeFor[typeTestStruct]()))
}

func TestCheckBindType(t *testing.T) {
	assert.NotPanics(t, func() { checkBindType(T[typeTestInterface]()) })
	assert.NotPanics(t, func() { checkBindType(T[*typeTestStruct]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[map[string]int]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[[]string]()) })
	assert.NotPanics(t, func() { checkBindType(reflect.TypeFor[func()]()) })

	assert.Panics(t, func() { checkBindType(reflect.TypeFor[typeTestStruct]()) })
}

func TestParseResolveTargetType(t *testing.T) {
	var structPtr *typeTestStruct
	assert.Equal(t, T[*typeTestStruct](), parseResolveTargetType(&structPtr))

	var interfaceValue typeTestInterface
	assert.Equal(t, T[typeTestInterface](), parseResolveTargetType(&interfaceValue))
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
