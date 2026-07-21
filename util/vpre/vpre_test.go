package vpre

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type sampleStruct struct{}

func TestPanicAndPanicf(t *testing.T) {
	baseErr := errors.New("boom")

	assert.PanicsWithValue(t, baseErr, func() {
		Panic(baseErr)
	})
	assert.PanicsWithError(t, "panic: 7", func() {
		Panicf("panic: %d", 7)
	})
}

func TestCheckFunctions(t *testing.T) {
	var typedNil *sampleStruct
	notNil := &sampleStruct{}
	dict := map[string]int{"a": 1}

	assert.NotPanics(t, func() {
		Check(true, "unexpected")
		CheckNot(false, "unexpected")
		CheckOK(dict, "a", "unexpected")
		CheckNotOK(dict, "b", "unexpected")
		CheckNil(nil, "unexpected")
		CheckNil(typedNil, "unexpected")
		CheckNotNil(notNil, "unexpected")
		CheckEmpty("", "unexpected")
		CheckNotEmpty("value", "unexpected")
	})

	assert.PanicsWithError(t, "condition failed", func() {
		Check(false, "condition failed")
	})
	assert.PanicsWithError(t, "condition must be false", func() {
		CheckNot(true, "condition must be false")
	})
	assert.PanicsWithError(t, "missing key", func() {
		CheckOK(dict, "b", "missing key")
	})
	assert.PanicsWithError(t, "duplicate key", func() {
		CheckNotOK(dict, "a", "duplicate key")
	})
	assert.PanicsWithError(t, "must be nil", func() {
		CheckNil(notNil, "must be nil")
	})
	assert.PanicsWithError(t, "must not be nil", func() {
		CheckNotNil(typedNil, "must not be nil")
	})
	assert.PanicsWithError(t, "must be empty", func() {
		CheckEmpty("value", "must be empty")
	})
	assert.PanicsWithError(t, "must not be empty", func() {
		CheckNotEmpty("", "must not be empty")
	})
	assert.PanicsWithError(t, "lazy message", func() {
		CheckFunc(false, func() string {
			return "lazy message"
		})
	})
}

func TestCheckNilErrorVariants(t *testing.T) {
	errBoom := errors.New("boom")
	actionCalled := false
	var actionErr error

	assert.NotPanics(t, func() {
		CheckNilError(nil, "no error")
		CheckNilErrorWithAction(nil, func(error) {
			actionCalled = true
		}, "no error")
	})
	assert.False(t, actionCalled)

	assert.PanicsWithError(t, "load failed: boom", func() {
		CheckNilError(errBoom, "load failed")
	})

	assert.PanicsWithError(t, "save failed: boom", func() {
		CheckNilErrorWithAction(errBoom, func(err error) {
			actionCalled = true
			actionErr = err
		}, "save failed")
	})
	assert.True(t, actionCalled)
	assert.Equal(t, errBoom, actionErr)
}

func TestMustFunctions(t *testing.T) {
	var typedNil *sampleStruct
	notNil := &sampleStruct{}

	assert.NotPanics(t, func() {
		Must(true)
		MustNil(nil)
		MustNil(typedNil)
		MustNotNil(notNil)
		MustEmpty("")
		MustNotEmpty("value")
	})

	assert.PanicsWithValue(t, unexpectedCondition, func() {
		Must(false)
	})
	assert.PanicsWithValue(t, unexpectedCondition, func() {
		MustNotReach()
	})
	assert.PanicsWithValue(t, unexpectedCondition, func() {
		MustNil(notNil)
	})
	assert.PanicsWithValue(t, unexpectedCondition, func() {
		MustNotNil(typedNil)
	})
	assert.PanicsWithValue(t, unexpectedCondition, func() {
		MustEmpty("value")
	})
	assert.PanicsWithValue(t, unexpectedCondition, func() {
		MustNotEmpty("")
	})
}
