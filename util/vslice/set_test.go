package vslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcat(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3, 4}, Concat([]int{1, 2}, []int{3}, nil, []int{4}))
}

func TestIntersectUnionAndDifference(t *testing.T) {
	list1 := []int{1, 3, 5, 7, 9}
	list2 := []int{7, 9, 11, 13, 15}

	assert.Equal(t, []int{7, 9}, Intersect(list1, list2))
	assert.Equal(t, []int{1, 3, 5, 7, 9, 11, 13, 15}, Union(list1, list2))

	left, right := Difference(list1, list2)
	assert.Equal(t, []int{1, 3, 5}, left)
	assert.Equal(t, []int{11, 13, 15}, right)
}

func TestEqualVariants(t *testing.T) {
	assert.True(t, Equal([]int{1, 2, 3}, []int{1, 2, 3}))
	assert.False(t, Equal([]int{1, 2, 3}, []int{3, 2, 1}))
	assert.True(t, EqualDisorderly([]int{1, 2, 3}, []int{3, 2, 1}))
	assert.False(t, EqualDisorderly([]int{1, 2, 3}, []int{1, 2, 4}))
}
