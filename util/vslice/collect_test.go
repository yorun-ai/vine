package vslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	result := Collect(func(yield func(int) bool) {
		for _, value := range []int{1, 2, 3} {
			if !yield(value) {
				return
			}
		}
	})

	assert.Equal(t, []int{1, 2, 3}, result)
}
