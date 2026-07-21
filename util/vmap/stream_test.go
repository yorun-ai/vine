package vmap

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForEach(t *testing.T) {
	dict := map[string]int{"a": 1, "b": 2, "c": 3}

	var sum int
	var visited []string
	ForEach(dict, func(key string, value int) {
		sum += value
		visited = append(visited, key)
	})

	slices.Sort(visited)
	assert.Equal(t, []string{"a", "b", "c"}, visited)
	assert.Equal(t, 6, sum)
}

func TestFilter(t *testing.T) {
	dict := map[string]int{"a": 1, "b": 2, "c": 3}

	filtered := Filter(dict, func(_ string, value int) bool {
		return value%2 == 1
	})

	assert.Equal(t, map[string]int{"a": 1, "c": 3}, filtered)
}

func TestMapAndMapToSlice(t *testing.T) {
	dict := map[string]int{"a": 1, "b": 2}

	mapped := Map(dict, func(key string, value int) (string, int) {
		return key + key, value * 10
	})
	assert.Equal(t, map[string]int{"aa": 10, "bb": 20}, mapped)

	filteredMapped := FilteredMap(dict, func(key string, value int) (string, int, bool) {
		return key + key, value * 10, value%2 == 1
	})
	assert.Equal(t, map[string]int{"aa": 10}, filteredMapped)

	result := MapToSlice(dict, func(key string, value int) string {
		return key + ":" + string(rune('0'+value))
	})
	slices.Sort(result)
	assert.Equal(t, []string{"a:1", "b:2"}, result)

	filteredResult := FilteredMapToSlice(dict, func(key string, value int) (string, bool) {
		return key + ":" + string(rune('0'+value)), value%2 == 1
	})
	slices.Sort(filteredResult)
	assert.Equal(t, []string{"a:1"}, filteredResult)
}
