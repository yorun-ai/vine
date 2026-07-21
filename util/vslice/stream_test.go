package vslice

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForEachAndForEachI(t *testing.T) {
	source := []int{1, 3, 5}

	var squares []int
	ForEach(source, func(elem int) {
		squares = append(squares, elem*elem)
	})
	assert.Equal(t, []int{1, 9, 25}, squares)

	var indexed []int
	ForEachI(source, func(elem int, index int) {
		indexed = append(indexed, elem+index)
	})
	assert.Equal(t, []int{1, 4, 7}, indexed)
}

func TestFilterAndFilterI(t *testing.T) {
	source := []int{1, 2, 3, 4, 5}

	assert.Equal(t, []int{2, 4}, Filter(source, func(elem int) bool {
		return elem%2 == 0
	}))
	assert.Equal(t, []int{1, 3, 5}, FilterI(source, func(elem int, index int) bool {
		return index%2 == 0 && elem%2 == 1
	}))
}

func TestMapVariants(t *testing.T) {
	source := []int{1, 3, 5}

	assert.Equal(t, []string{"v=1", "v=3", "v=5"}, Map(source, func(elem int) string {
		return fmt.Sprintf("v=%d", elem)
	}))
	assert.Equal(t, []string{"[0]=1", "[1]=3", "[2]=5"}, MapI(source, func(elem int, index int) string {
		return fmt.Sprintf("[%d]=%d", index, elem)
	}))
	assert.Equal(t, []string{"odd-1", "odd-3", "odd-5"}, FilteredMap(source, func(elem int) (string, bool) {
		return fmt.Sprintf("odd-%d", elem), elem%2 == 1
	}))
	assert.Equal(t, []string{"1@0", "5@2"}, FilteredMapI(source, func(elem int, index int) (string, bool) {
		return fmt.Sprintf("%d@%d", elem, index), index%2 == 0
	}))
}

func TestMapToMapAndReduce(t *testing.T) {
	source := []int{1, 3, 5}

	assert.Equal(t, map[string]int{
		"1": 1,
		"3": 3,
		"5": 5,
	}, MapToMap(source, func(elem int) (string, int) {
		return fmt.Sprintf("%d", elem), elem
	}))

	assert.Equal(t, map[int]string{
		0: "1",
		1: "3",
		2: "5",
	}, MapToMapI(source, func(elem int, index int) (int, string) {
		return index, fmt.Sprintf("%d", elem)
	}))

	assert.Equal(t, 9, Reduce(source, func(total int, elem int) int {
		return total + elem
	}, 0))
	assert.Equal(t, "0:1|1:3|2:5", ReduceI(source, func(result string, elem int, index int) string {
		if result != "" {
			result += "|"
		}
		return fmt.Sprintf("%s%d:%d", result, index, elem)
	}, ""))
}
