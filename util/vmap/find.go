package vmap

import "slices"

// All returns true if all entries meets prediction
func All[K comparable, V any](dict map[K]V, predicate func(K, V) bool) bool {
	for k, v := range dict {
		if !predicate(k, v) {
			return false
		}
	}
	return true
}

// Any returns true if at least 1 entry meets prediction
func Any[K comparable, V any](dict map[K]V, predicate func(K, V) bool) bool {
	for k, v := range dict {
		if predicate(k, v) {
			return true
		}
	}
	return false
}

// None returns true if no entry meets prediction
func None[K comparable, V any](dict map[K]V, predicate func(K, V) bool) bool {
	for k, v := range dict {
		if predicate(k, v) {
			return false
		}
	}
	return true
}

// ContainsKey returns true if dict contains the key.
func ContainsKey[K comparable, V any](dict map[K]V, key K) bool {
	_, ok := dict[key]
	return ok
}

// ContainsAllKey returns true if dict contains all keys.
func ContainsAllKey[K comparable, V any](dict map[K]V, keys []K) bool {
	for _, key := range keys {
		if _, ok := dict[key]; !ok {
			return false
		}
	}
	return true
}

// ContainsAnyKey returns true if dict contains at least 1 key.
func ContainsAnyKey[K comparable, V any](dict map[K]V, keys []K) bool {
	return slices.ContainsFunc(keys, func(key K) bool {
		_, ok := dict[key]
		return ok
	})
}

// ContainsNoneKey returns true if dict do not contain any key.
func ContainsNoneKey[K comparable, V any](dict map[K]V, keys []K) bool {
	return !ContainsAnyKey(dict, keys)
}
