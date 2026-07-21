package vslice

// ForEach iterates over elements of list and invokes iteratee for each element.
func ForEach[E any](list []E, iteratee func(E)) {
	ForEachI(list, func(elem E, _ int) {
		iteratee(elem)
	})
}

// ForEachI do ForEach with index.
func ForEachI[E any](list []E, iteratee func(E, int)) {
	for i, elem := range list {
		iteratee(elem, i)
	}
}

// Filter iterates over elements of slice, returning an array of all elements predicate returns truthy for.
func Filter[E any](list []E, predicate func(E) bool) []E {
	return FilterI(list, func(elem E, _ int) bool {
		return predicate(elem)
	})
}

// FilterI do Filter with index.
func FilterI[E any](list []E, predicate func(E, int) bool) []E {
	result := []E{}
	for i, elem := range list {
		if predicate(elem, i) {
			result = append(result, elem)
		}
	}
	return result
}

// Map manipulates a slice and transforms it to a slice of another type.
func Map[E any, R any](list []E, iteratee func(E) R) []R {
	return MapI(list, func(elem E, _ int) R {
		return iteratee(elem)
	})
}

// MapI do Map with index.
func MapI[E any, R any](list []E, iteratee func(E, int) R) []R {
	result := make([]R, len(list))
	for i, elem := range list {
		result[i] = iteratee(elem, i)
	}
	return result
}

// FilteredMap do Map with filter.
func FilteredMap[E any, R any](list []E, iteratee func(E) (R, bool)) []R {
	return FilteredMapI(list, func(elem E, _ int) (R, bool) {
		return iteratee(elem)
	})
}

// FilteredMapI do FilteredMap with index.
func FilteredMapI[E any, R any](list []E, iteratee func(E, int) (R, bool)) []R {
	result := make([]R, 0)
	for i, elem := range list {
		if mapped, ok := iteratee(elem, i); ok {
			result = append(result, mapped)
		}
	}
	return result
}

// MapToMap returns a map containing key-value pairs provided by transform function applied to elements of the given slice.
// If any of two pairs would have the same key the last one gets added to the map.
func MapToMap[E any, K comparable, V any](list []E, transform func(E) (K, V)) map[K]V {
	return MapToMapI(list, func(elem E, _ int) (K, V) {
		return transform(elem)
	})
}

// MapToMapI do MapToMap with index.
func MapToMapI[E any, K comparable, V any](list []E, transform func(E, int) (K, V)) map[K]V {
	result := make(map[K]V)
	for i, elem := range list {
		key, value := transform(elem, i)
		result[key] = value
	}
	return result
}

// Reduce reduces list to a value which is the accumulated result of running each element in list
// through accumulator, where each successive invocation is supplied the return value of the previous.
func Reduce[E any, R any](list []E, accumulator func(R, E) R, initial R) R {
	return ReduceI(list, func(result R, elem E, _ int) R {
		return accumulator(result, elem)
	}, initial)
}

// ReduceI do Reduce with index.
func ReduceI[E any, R any](list []E, accumulator func(R, E, int) R, initial R) R {
	for i, elem := range list {
		initial = accumulator(initial, elem, i)
	}
	return initial
}
