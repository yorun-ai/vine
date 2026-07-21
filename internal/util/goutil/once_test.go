package goutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverableOnce(t *testing.T) {
	var once RecoverableOnce
	count := 0

	assert.PanicsWithValue(t, "boom", func() {
		once.Do(func() {
			count++
			panic("boom")
		})
	})

	assert.Equal(t, 1, count)

	assert.NotPanics(t, func() {
		once.Do(func() {
			count++
		})
	})
	assert.Equal(t, 2, count)

	assert.NotPanics(t, func() {
		once.Do(func() {
			count++
		})
	})
	assert.Equal(t, 2, count)
}
