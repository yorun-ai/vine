package vmap

import "maps"

// ForEach iterates over entries of map and invokes iteratee for each entry.
func ForEach[K comparable, V any](dict map[K]V, iteratee func(K, V)) {
	for k, v := range maps.All(dict) {
		iteratee(k, v)
	}
}

// Filter iterates over entries of map, returning a map of all entries predicate returns truthy for.
func Filter[K comparable, V any](dict map[K]V, predicate func(K, V) bool) map[K]V {
	result := maps.Clone(dict)
	maps.DeleteFunc(result, func(k K, v V) bool {
		return !predicate(k, v)
	})
	return result
}

// Map manipulates a map and transforms it to another map.
func Map[K1, K2 comparable, V1, V2 any](dict map[K1]V1, transform func(K1, V1) (K2, V2)) map[K2]V2 {
	result := map[K2]V2{}
	for k1, v1 := range dict {
		k2, v2 := transform(k1, v1)
		result[k2] = v2
	}
	return result
}

// FilteredMap does Map with filter.
func FilteredMap[K1, K2 comparable, V1, V2 any](dict map[K1]V1, transform func(K1, V1) (K2, V2, bool)) map[K2]V2 {
	result := map[K2]V2{}
	for k1, v1 := range dict {
		if k2, v2, ok := transform(k1, v1); ok {
			result[k2] = v2
		}
	}
	return result
}

// MapToSlice manipulates a map and transforms it to a slice
func MapToSlice[K comparable, V any, E any](dict map[K]V, transform func(K, V) E) []E {
	result := []E{}
	for k, v := range dict {
		result = append(result, transform(k, v))
	}
	return result
}

// FilteredMapToSlice does MapToSlice with filter.
func FilteredMapToSlice[K comparable, V any, E any](dict map[K]V, transform func(K, V) (E, bool)) []E {
	result := []E{}
	for k, v := range dict {
		if mapped, ok := transform(k, v); ok {
			result = append(result, mapped)
		}
	}
	return result
}
