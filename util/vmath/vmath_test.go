package vmath

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInRange(t *testing.T) {
	assert.True(t, InRange(5, 1, 10))
	assert.False(t, InRange(10, 1, 10))
	assert.True(t, InRange(5*time.Second, time.Second, 10*time.Second))
}

func TestRandIntBetween(t *testing.T) {
	for range 50 {
		value := RandIntBetween(3, 5)
		assert.GreaterOrEqual(t, value, 3)
		assert.LessOrEqual(t, value, 5)
	}

	assert.Panics(t, func() {
		RandIntBetween(5, 5)
	})
}
