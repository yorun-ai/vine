package vmap

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentMaps(t *testing.T) {
	type concurrentMap interface {
		Load(string) (int, bool)
		Store(string, int)
		LoadOrStore(string, int) (int, bool)
		LoadAndDelete(string) (int, bool)
		Delete(string)
		Range(func(string, int) bool)
	}
	for _, tt := range []struct {
		name string
		dict concurrentMap
	}{
		{"MutexMap", NewMutexMap[string, int]()},
		{"SyncMap", NewSyncMap[string, int]()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dict := tt.dict
			value, ok := dict.Load("missing")
			assert.False(t, ok)
			assert.Zero(t, value)

			dict.Store("a", 1)
			value, ok = dict.Load("a")
			assert.True(t, ok)
			assert.Equal(t, 1, value)

			actual, loaded := dict.LoadOrStore("b", 2)
			assert.False(t, loaded)
			assert.Equal(t, 2, actual)
			actual, loaded = dict.LoadOrStore("b", 3)
			assert.True(t, loaded)
			assert.Equal(t, 2, actual)

			visits := 0
			dict.Range(func(string, int) bool {
				visits++
				return false
			})
			assert.Equal(t, 1, visits)
			var keys []string
			dict.Range(func(key string, _ int) bool {
				keys = append(keys, key)
				return true
			})
			slices.Sort(keys)
			assert.Equal(t, []string{"a", "b"}, keys)

			value, ok = dict.LoadAndDelete("a")
			assert.True(t, ok)
			assert.Equal(t, 1, value)
			_, ok = dict.Load("a")
			assert.False(t, ok)
			dict.Delete("b")
			_, ok = dict.Load("b")
			assert.False(t, ok)
		})
	}
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
