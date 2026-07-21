package reflectutil

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPointerHelpers(t *testing.T) {
	pointed := PointerTo(7)
	assert.Equal(t, 7, *pointed)
}

func TestStructHelpers(t *testing.T) {
	item := holder{
		EmbeddedA: EmbeddedA{A: 1},
		EmbeddedB: EmbeddedB{B: "two"},
		Visible:   "three",
		hidden:    4,
	}

	assert.Equal(t, 4, GetPrivateField(&item, "hidden"))

	exported := exportedHolder{
		EmbeddedA: EmbeddedA{A: 1},
		EmbeddedB: EmbeddedB{B: "two"},
		Visible:   "three",
	}

	fields := FlattenFields(exported)
	assert.Len(t, fields, 3)
	assert.Equal(t, 1, fields[0].Interface())
	assert.Equal(t, "two", fields[1].Interface())
	assert.Equal(t, "three", fields[2].Interface())

	embeddedTypes := EmbeddedStructTypes(reflect.TypeOf(exported))
	assert.Equal(t, []reflect.Type{reflect.TypeOf(EmbeddedA{}), reflect.TypeOf(EmbeddedB{})}, embeddedTypes)

	embeddedFields := EmbeddedStructFields(reflect.ValueOf(exported))
	assert.Len(t, embeddedFields, 2)
	assert.Equal(t, reflect.TypeOf(EmbeddedA{}), embeddedFields[0].Type())
	assert.Equal(t, reflect.TypeOf(EmbeddedB{}), embeddedFields[1].Type())
}

func TestUnsafeFiled(t *testing.T) {
	item := holder{hidden: 1}
	field := reflect.ValueOf(&item).Elem().FieldByName("hidden")
	unsafeField := UnsafeField(field)
	unsafeField.SetInt(9)

	assert.Equal(t, 9, item.hidden)
}

func TestCloneStructPointer(t *testing.T) {
	source := &exportedHolder{
		EmbeddedA: EmbeddedA{A: 1},
		EmbeddedB: EmbeddedB{B: "two"},
		Visible:   "three",
	}

	cloned, ok := CloneStructPointer(source).(*exportedHolder)
	if assert.True(t, ok) {
		assert.Equal(t, source, cloned)
		assert.NotSame(t, source, cloned)
	}
}
