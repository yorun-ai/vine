package cacheutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLruSetRejectsInvalidCapacity(t *testing.T) {
	assert.Panics(t, func() {
		NewLruSet[string](0)
	})
}

func TestLruSetEvictsLeastRecentlyUsedElement(t *testing.T) {
	set := NewLruSet[string](2)
	set.Add("first")
	set.Add("second")
	assert.True(t, set.Contains("first"))

	set.Add("third")

	assert.True(t, set.Contains("first"))
	assert.False(t, set.Contains("second"))
	assert.True(t, set.Contains("third"))
	assert.Equal(t, 2, set.Len())
}

func TestLruSetAddRefreshesExistingElement(t *testing.T) {
	set := NewLruSet[string](2)
	set.Add("first")
	set.Add("second")
	set.Add("first")

	set.Add("third")

	assert.True(t, set.Contains("first"))
	assert.False(t, set.Contains("second"))
	assert.True(t, set.Contains("third"))
}
