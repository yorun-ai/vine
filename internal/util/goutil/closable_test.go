package goutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testClosable struct {
	called bool
	err    error
}

func (c *testClosable) Close() error {
	c.called = true
	return c.err
}

func TestClose(t *testing.T) {
	okClosable := &testClosable{}
	assert.NotPanics(t, func() {
		Close(okClosable)
	})
	assert.True(t, okClosable.called)

	errClosable := &testClosable{err: errors.New("close failed")}
	assert.NotPanics(t, func() {
		Close(errClosable)
	})
	assert.True(t, errClosable.called)
}
