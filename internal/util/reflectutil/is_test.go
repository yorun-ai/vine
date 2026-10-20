package reflectutil

import (
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

func TestNilHelpers(t *testing.T) {
	var ptr *holder
	assert.True(t, IsNil(ptr))
	assert.True(t, IsNilValue(reflect.ValueOf(ptr)))
	assert.True(t, IsNil(nil))
	assert.False(t, IsNil(0))
	assert.False(t, IsNil(holder{}))
	assert.False(t, IsNil(""))
}
