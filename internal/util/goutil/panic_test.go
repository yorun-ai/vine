package goutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnPanicHelpers(t *testing.T) {
	for name, run := range map[string]func(func(), func()){
		"wrapped": func(fn func(), onPanic func()) { WrapWithOnPanic(fn, onPanic)() },
		"direct":  RunWithOnPanic,
	} {
		t.Run(name, func(t *testing.T) {
			called, panicHandled := false, false
			onPanic := func() { panicHandled = true }
			run(func() { called = true }, onPanic)
			assert.True(t, called)
			assert.False(t, panicHandled)

			assert.PanicsWithValue(t, "boom", func() {
				run(func() { panic("boom") }, onPanic)
			})
			assert.True(t, panicHandled)
		})
	}
}

func TestRunWithRecover(t *testing.T) {
	t.Run("passes arguments to target function", func(t *testing.T) {
		var got int

		RunWithRecover(func(any) {
			t.Fatal("recover handler should not be called")
		}, func(a int, b int) {
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
