package vmap

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutexMapStoreLoadAndDelete(t *testing.T) {
	dict := NewMutexMap[string, int]()

	value, ok := dict.Load("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, value)

	dict.Store("a", 1)

	value, ok = dict.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 1, value)

	deleted, ok := dict.LoadAndDelete("a")
	assert.True(t, ok)
	assert.Equal(t, 1, deleted)

	_, ok = dict.Load("a")
	assert.False(t, ok)
}

func TestMutexMapLoadOrStore(t *testing.T) {
	dict := NewMutexMap[string, int]()

	actual, loaded := dict.LoadOrStore("a", 1)
	assert.False(t, loaded)
	assert.Equal(t, 1, actual)

	actual, loaded = dict.LoadOrStore("a", 2)
	assert.True(t, loaded)
	assert.Equal(t, 1, actual)
}

func TestMutexMapRange(t *testing.T) {
	dict := NewMutexMap[string, int]()
	dict.Store("a", 1)
	dict.Store("b", 2)

	var keys []string
	dict.Range(func(key string, value int) bool {
		keys = append(keys, key)
		return false
	})

	assert.Len(t, keys, 1)
	keys = nil

	dict.Range(func(key string, _ int) bool {
		keys = append(keys, key)
		return true
	})

	slices.Sort(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}

func TestMutexMapRangeAllowsMutationInCallback(t *testing.T) {
	dict := NewMutexMap[string, int]()
	dict.Store("a", 1)

	dict.Range(func(key string, _ int) bool {
		dict.Store(key, 2)
		return true
	})

	value, ok := dict.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 2, value)
}

func TestMutexMapDelete(t *testing.T) {
	dict := NewMutexMap[string, int]()
	dict.Store("a", 1)
	dict.Delete("a")

	_, ok := dict.Load("a")
	assert.False(t, ok)
}

func TestSyncMapStoreLoadAndDelete(t *testing.T) {
	dict := NewSyncMap[string, int]()

	value, ok := dict.Load("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, value)

	dict.Store("a", 1)

	value, ok = dict.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 1, value)

	deleted, ok := dict.LoadAndDelete("a")
	assert.True(t, ok)
	assert.Equal(t, 1, deleted)

	_, ok = dict.Load("a")
	assert.False(t, ok)
}

func TestSyncMapLoadOrStore(t *testing.T) {
	dict := NewSyncMap[string, int]()

	actual, loaded := dict.LoadOrStore("a", 1)
	assert.False(t, loaded)
	assert.Equal(t, 1, actual)

	actual, loaded = dict.LoadOrStore("a", 2)
	assert.True(t, loaded)
	assert.Equal(t, 1, actual)
}

func TestSyncMapRange(t *testing.T) {
	dict := NewSyncMap[string, int]()
	dict.Store("a", 1)
	dict.Store("b", 2)

	var keys []string
	dict.Range(func(key string, value int) bool {
		keys = append(keys, key)
		return false
	})

	assert.Len(t, keys, 1)
	keys = nil

	dict.Range(func(key string, _ int) bool {
		keys = append(keys, key)
		return true
	})

	slices.Sort(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}

func TestSyncMapDelete(t *testing.T) {
	dict := NewSyncMap[string, int]()
	dict.Store("a", 1)
	dict.Delete("a")

	_, ok := dict.Load("a")
	assert.False(t, ok)
}
