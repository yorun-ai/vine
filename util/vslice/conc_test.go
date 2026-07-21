package vslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutexSliceAppendLoadStoreAndLen(t *testing.T) {
	list := NewMutexSlice(1, 2)

	assert.Equal(t, 2, list.Len())

	value, ok := list.Load(1)
	assert.True(t, ok)
	assert.Equal(t, 2, value)

	assert.True(t, list.Store(1, 20))
	value, ok = list.Load(1)
	assert.True(t, ok)
	assert.Equal(t, 20, value)

	list.Append(3, 4)
	assert.Equal(t, 4, list.Len())

	value, ok = list.Load(-1)
	assert.False(t, ok)
	assert.Equal(t, 0, value)

	assert.False(t, list.Store(9, 90))
}

func TestMutexSliceDelete(t *testing.T) {
	list := NewMutexSlice("a", "b", "c")

	value, ok := list.Delete(1)
	assert.True(t, ok)
	assert.Equal(t, "b", value)
	assert.Equal(t, []string{"a", "c"}, list.Snapshot())

	value, ok = list.Delete(9)
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestMutexSliceSnapshotReturnsCopy(t *testing.T) {
	list := NewMutexSlice(1, 2, 3)

	snapshot := list.Snapshot()
	snapshot[0] = 9

	assert.Equal(t, []int{1, 2, 3}, list.Snapshot())
}

func TestMutexSliceRange(t *testing.T) {
	list := NewMutexSlice(1, 2, 3)

	var values []int
	list.Range(func(_ int, item int) bool {
		values = append(values, item)
		return item != 2
	})

	assert.Equal(t, []int{1, 2}, values)
}

func TestMutexSliceRangeAllowsMutationInCallback(t *testing.T) {
	list := NewMutexSlice(1)

	list.Range(func(_ int, item int) bool {
		list.Append(item + 1)
		return true
	})

	assert.Equal(t, []int{1, 2}, list.Snapshot())
}
