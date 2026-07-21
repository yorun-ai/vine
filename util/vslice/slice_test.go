package vslice

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloneUniqueAndUniqueBy(t *testing.T) {
	original := []int{1, 1, 3, 5, 7, 7, 9}
	cloned := Clone(original)
	cloned[0] = 99

	assert.Equal(t, []int{1, 1, 3, 5, 7, 7, 9}, original)
	assert.Equal(t, []int{1, 3, 5, 7, 9}, Unique(original))
	assert.Equal(t, []string{"a", "aa", "ddd", "ddddd"}, UniqueBy([]string{"a", "aa", "bb", "c", "ddd", "ddddd"}, func(elem string) int {
		return len(elem)
	}))
	assert.Equal(t, []reflect.Type{
		reflect.TypeOf(1),
		reflect.TypeOf(""),
	}, Unique([]reflect.Type{
		reflect.TypeOf(1),
		reflect.TypeOf(1),
		reflect.TypeOf(""),
	}))
	assert.Equal(t, []int{1, 4}, Delete([]int{1, 2, 3, 4}, 1, 3))
}

func TestGroupByAndFlatten(t *testing.T) {
	grouped := GroupBy([]string{"ant", "bear", "cat", "dog"}, func(elem string) int {
		return len(elem)
	})

	assert.Equal(t, map[int][]string{
		3: {"ant", "cat", "dog"},
		4: {"bear"},
	}, grouped)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, Flatten([][]int{{1, 2}, nil, {3, 4, 5}}))
}

func TestRepeatVariantsAndChunk(t *testing.T) {
	assert.Equal(t, []string{"x", "x", "x"}, Repeat(3, "x"))
	counter := 0
	assert.Equal(t, []int{1, 2, 3}, RepeatBy(3, func() int {
		counter++
		return counter
	}))
	assert.Equal(t, []string{"0", "1", "2"}, RepeatByI(3, func(index int) string {
		return string(rune('0' + index))
	}))
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, Chunk([]int{1, 2, 3, 4, 5}, 2))
	assert.PanicsWithValue(t, "Second parameter must be greater than 0", func() {
		Chunk([]int{1, 2, 3}, 0)
	})
}

func TestShuffleReverseAndSort(t *testing.T) {
	original := []int{9, 7, 1, 3, 5}

	shuffled := Shuffle(original)
	assert.Equal(t, len(original), len(shuffled))
	assert.True(t, EqualDisorderly(original, shuffled))
	assert.Equal(t, []int{9, 7, 1, 3, 5}, original)

	assert.Equal(t, []int{5, 3, 1, 7, 9}, Reverse([]int{9, 7, 1, 3, 5}))
	assert.Equal(t, []string{"b", "ee", "aaa", "cccc", "ddddd"}, SortBy([]string{"aaa", "b", "cccc", "ddddd", "ee"}, func(left string, right string) bool {
		return len(left) < len(right)
	}))
	assert.Equal(t, []int{1, 3, 5, 7, 9}, Sort(original))
	assert.Equal(t, []int{9, 7, 5, 3, 1}, SortDesc(original))
	assert.Equal(t, []int{9, 7, 1, 3, 5}, original)
}
