package goutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapWithOnPanic(t *testing.T) {
	t.Run("runs function without panic handler when no panic occurs", func(t *testing.T) {
		called := false
		panicHandled := false

		WrapWithOnPanic(func() {
			called = true
		}, func() {
			panicHandled = true
		})()

		assert.True(t, called)
		assert.False(t, panicHandled)
	})

	t.Run("invokes panic handler and preserves panic", func(t *testing.T) {
		panicHandled := false

		assert.PanicsWithValue(t, "boom", func() {
			WrapWithOnPanic(func() {
				panic("boom")
			}, func() {
				panicHandled = true
			})()
		})

		assert.True(t, panicHandled)
	})
}

func TestRunWithOnPanic(t *testing.T) {
	panicHandled := false

	assert.PanicsWithValue(t, "boom", func() {
		RunWithOnPanic(func() {
			panic("boom")
		}, func() {
			panicHandled = true
		})
	})

	assert.True(t, panicHandled)
}

func TestRunWithRecover(t *testing.T) {
	t.Run("passes arguments to target function", func(t *testing.T) {
		var got int

		RunWithRecover(func(any) {
			t.Fatal("recover handler should not be called")
		}, func(a, b int) {
			got = a + b
		}, 2, 3)

		assert.Equal(t, 5, got)
	})

	t.Run("forwards recovered panic", func(t *testing.T) {
		panicValue := errors.New("boom")
		var recovered any

		RunWithRecover(func(r any) {
			recovered = r
		}, func() {
			panic(panicValue)
		})

		assert.Equal(t, panicValue, recovered)
	})
}
