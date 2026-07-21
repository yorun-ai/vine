package vslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredicatesAndContains(t *testing.T) {
	list := []int{1, 2, 3, 4, 5}

	assert.True(t, All(list, func(elem int) bool { return elem > 0 }))
	assert.False(t, All(list, func(elem int) bool { return elem < 5 }))
	assert.True(t, Any(list, func(elem int) bool { return elem == 4 }))
	assert.False(t, Any(list, func(elem int) bool { return elem > 5 }))
	assert.True(t, None(list, func(elem int) bool { return elem < 0 }))
	assert.False(t, None(list, func(elem int) bool { return elem == 5 }))
	assert.True(t, Contains(list, 3))
	assert.False(t, Contains(list, 9))
	assert.True(t, ContainsAll(list, []int{1, 3, 5}))
	assert.False(t, ContainsAll(list, []int{1, 6}))
	assert.True(t, ContainsAny(list, []int{6, 5}))
	assert.False(t, ContainsAny(list, []int{6, 7}))
	assert.True(t, ContainsNone(list, []int{6, 7}))
	assert.False(t, ContainsNone(list, []int{6, 2}))
}

func TestIndexAndFindHelpers(t *testing.T) {
	list := []int{1, 2, 3, 4, 5}

	assert.Equal(t, 2, Index(list, 3))
	assert.Equal(t, -1, Index(list, 9))
	assert.Equal(t, 3, IndexBy(list, func(elem int) bool { return elem%4 == 0 }))
	assert.Equal(t, -1, IndexBy(list, func(elem int) bool { return elem < 0 }))

	elem, ok := Find(list, func(elem int) bool { return elem > 3 })
	assert.True(t, ok)
	assert.Equal(t, 4, elem)

	elem, index, ok := FindI(list, func(elem int) bool { return elem%2 == 0 })
	assert.True(t, ok)
	assert.Equal(t, 2, elem)
	assert.Equal(t, 1, index)

	missingElem, missingIndex, missingOK := FindI(list, func(elem int) bool { return elem < 0 })
	assert.False(t, missingOK)
	assert.Zero(t, missingElem)
	assert.Equal(t, -1, missingIndex)

	assert.Equal(t, 5, FindOrElse(list, 99, func(elem int) bool { return elem == 5 }))
	assert.Equal(t, 99, FindOrElse(list, 99, func(elem int) bool { return elem == 10 }))
}

func TestMostByMinAndMax(t *testing.T) {
	words := []string{"aaaa", "b", "cccc", "ddddd", "ee"}

	assert.Equal(t, "b", MostBy(words, func(left string, right string) bool {
		return len(left) < len(right)
	}))
	assert.Equal(t, "ddddd", MostBy(words, func(left string, right string) bool {
		return len(left) > len(right)
	}))
	assert.Equal(t, -999, Min([]int{100, 300, -200, -999, 40, 99}))
	assert.Equal(t, 999, Max([]int{-100, 500, 600, 999, 700}))
	assert.Zero(t, MostBy([]string(nil), func(left string, right string) bool {
		return left < right
	}))
}
