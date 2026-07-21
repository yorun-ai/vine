package vmap

import (
	"maps"
	"slices"

	"golang.org/x/exp/constraints"
)

// Clone returns a copy of map.
func Clone[M ~map[K]V, K comparable, V any](m M) M {
	return maps.Clone(m)
}

// Copy copies all key/value pairs in src adding them to dst.
func Copy[M1 ~map[K]V, M2 ~map[K]V, K comparable, V any](dst M1, src M2) {
	maps.Copy(dst, src)
}

// Merge merges maps, the later map take high priority
func Merge[K comparable, V any](dicts ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, dict := range dicts {
		maps.Copy(result, dict)
	}
	return result
}

// Keys creates a list of the map keys.
func Keys[Map ~map[K]V, K comparable, V any](m Map) []K {
	return slices.Collect(maps.Keys(m))
}

// SortedKeys returns the map keys in ascending order.
func SortedKeys[Map ~map[K]V, K constraints.Ordered, V any](m Map) []K {
	return slices.Sorted(maps.Keys(m))
}

// Values creates a list of the map values.
func Values[Map ~map[K]V, K comparable, V any](m Map) []V {
	return slices.Collect(maps.Values(m))
}

// SortedValues returns values ordered by their corresponding ascending keys.
func SortedValues[Map ~map[K]V, K constraints.Ordered, V any](m Map) []V {
	keys := SortedKeys(m)
	values := make([]V, 0, len(keys))
	for _, key := range keys {
		values = append(values, m[key])
	}
	return values
}
