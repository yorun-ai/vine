package reflectutil

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type EmbeddedA struct {
	A int
}

type EmbeddedB struct {
	B string
}

type holder struct {
	EmbeddedA
	EmbeddedB
	Visible string
	hidden  int
}

type exportedHolder struct {
	EmbeddedA
	EmbeddedB
	Visible string
}

func sampleFunc() {}

func TestKindHelpers(t *testing.T) {
	assert.True(t, IsError(errors.New("boom")))
	assert.True(t, IsErrorType(reflect.TypeOf((*error)(nil)).Elem()))
	assert.True(t, IsInterfaceType(reflect.TypeOf((*error)(nil)).Elem()))
	assert.True(t, IsPointerType(reflect.TypeOf(&holder{})))
	assert.True(t, IsStructPointerType(reflect.TypeOf(&holder{})))
	assert.True(t, IsFunc(sampleFunc))
	assert.True(t, IsMap(map[string]int{}))
	assert.True(t, IsSlice([]int{}))
}

func TestNilHelpers(t *testing.T) {
	var ptr *holder
	assert.True(t, IsNil(ptr))
	assert.True(t, IsNilValue(reflect.ValueOf(ptr)))
	assert.True(t, IsNil(nil))
	assert.False(t, IsNil(0))
	assert.False(t, IsNil(holder{}))
	assert.False(t, IsNil(""))
}
