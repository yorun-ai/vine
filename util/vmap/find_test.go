package vmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredicatesAndContainsKey(t *testing.T) {
	dict := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	assert.True(t, All(dict, func(k int, v string) bool {
		return k > 0 && v != ""
	}))
	assert.False(t, All(dict, func(k int, _ string) bool {
		return k < 3
	}))
	assert.True(t, Any(dict, func(k int, _ string) bool {
		return k == 2
	}))
	assert.False(t, Any(dict, func(_ int, v string) bool {
		return v == "missing"
	}))
	assert.True(t, None(dict, func(k int, _ string) bool {
		return k < 0
	}))
	assert.False(t, None(dict, func(k int, _ string) bool {
		return k == 1
	}))
	assert.True(t, ContainsKey(dict, 2))
	assert.False(t, ContainsKey(dict, 4))
}

func TestContainsKeyCollections(t *testing.T) {
	dict := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	assert.True(t, ContainsAllKey(dict, []int{1, 3}))
	assert.False(t, ContainsAllKey(dict, []int{1, 4}))
	assert.True(t, ContainsAnyKey(dict, []int{4, 2}))
	assert.False(t, ContainsAnyKey(dict, []int{4, 5}))
	assert.True(t, ContainsNoneKey(dict, []int{4, 5}))
	assert.False(t, ContainsNoneKey(dict, []int{4, 2}))
}

func TestPredicatesWithEmptyMap(t *testing.T) {
	dict := map[int]string{}

	assert.True(t, All(dict, func(int, string) bool { return false }))
	assert.False(t, Any(dict, func(int, string) bool { return true }))
	assert.True(t, None(dict, func(int, string) bool { return true }))
	assert.True(t, ContainsAllKey(dict, nil))
	assert.True(t, ContainsNoneKey(dict, []int{1}))
}
