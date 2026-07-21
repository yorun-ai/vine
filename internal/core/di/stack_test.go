package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type stackTypeA struct{}
type stackTypeB struct{}

func TestBuildStackPush(t *testing.T) {
	stack := _BuildStack{T[*stackTypeA]()}

	next := stack.push(T[*stackTypeB]())

	assert.Equal(t, _BuildStack{T[*stackTypeA]()}, stack)
	assert.Equal(t, _BuildStack{T[*stackTypeA](), T[*stackTypeB]()}, next)
}

func TestBuildStackContains(t *testing.T) {
	stack := _BuildStack{T[*stackTypeA](), T[*stackTypeB]()}

	assert.True(t, stack.contains(T[*stackTypeA]()))
	assert.True(t, stack.contains(T[*stackTypeB]()))
	assert.False(t, stack.contains(T[Injector]()))
}

func TestBuildStackString(t *testing.T) {
	assert.Equal(t, "<empty>", _BuildStack(nil).String())

	stack := _BuildStack{T[*stackTypeA](), T[*stackTypeB]()}
	assert.Equal(t, "*di.stackTypeA -> *di.stackTypeB", stack.String())
}
