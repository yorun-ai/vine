package vslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	assert.Equal(t, 3, Count([]int{1, 2, 1, 3, 1}, 1))
	assert.Equal(t, 0, Count([]int{1, 2, 3}, 4))
}
