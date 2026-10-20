package vmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloneCopyAndMerge(t *testing.T) {
	source := map[string]int{"a": 1, "b": 2}

	cloned := Clone(source)
	cloned["a"] = 9
	assert.Equal(t, 1, source["a"])

	destination := map[string]int{"c": 3}
	Copy(destination, source)
	assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3}, destination)

	merged := Merge(
		map[string]int{"a": 1, "b": 2},
		map[string]int{"b": 4},
		map[string]int{"c": 5},
	)
	assert.Equal(t, map[string]int{"a": 1, "b": 4, "c": 5}, merged)
}
