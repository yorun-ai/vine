package vslice

import (
	"slices"

	"golang.org/x/exp/constraints"
)

// All returns true if all elements meets prediction.
func All[E any](list []E, predicate func(E) bool) bool {
	return !slices.ContainsFunc(list, func(elem E) bool {
		return !predicate(elem)
	})
}

// Any returns true if at least 1 element meets prediction.
func Any[E any](list []E, predicate func(E) bool) bool {
	return slices.ContainsFunc(list, predicate)
}

// None returns true if no element meets prediction.
func None[E any](list []E, predicate func(E) bool) bool {
	return !slices.ContainsFunc(list, predicate)
}

// Contains returns true if list contains the item.
func Contains[E comparable](list []E, item E) bool {
	return slices.Contains(list, item)
}

// ContainsAll returns true if list contains all item.
func ContainsAll[E comparable](list []E, items []E) bool {
	found := map[E]bool{}
	for _, item := range items {
		found[item] = false
	}
	for _, elem := range list {
		if _, exists := found[elem]; exists {
			found[elem] = true
		}
	}
	for _, result := range found {
		if !result {
			return false
		}
	}
	return true
}

// ContainsAny returns true if list contains at least 1 item.
func ContainsAny[E comparable](list []E, items []E) bool {
	return slices.ContainsFunc(items, func(item E) bool {
		return slices.Contains(list, item)
	})
}

// ContainsNone returns true if list do not contain any item.
func ContainsNone[E comparable](list []E, items []E) bool {
	return !ContainsAny(list, items)
}

// Index returns the index at which the first occurrence of a value is found in an array or return -1
// if the value cannot be found.
func Index[E comparable](list []E, item E) int {
	return slices.Index(list, item)
}

// IndexBy returns the index at which the first occurrence of an element matches the predication,
// returns -1 if all elements not matched.
func IndexBy[E any](list []E, predicate func(E) bool) int {
	return slices.IndexFunc(list, predicate)
}

// Find try to find the first element meets prediction.
func Find[E any](list []E, predicate func(E) bool) (E, bool) {
	elem, _, ok := FindI(list, predicate)
	return elem, ok
}

// FindI do Find with index returned.
func FindI[E any](list []E, predicate func(E) bool) (elem E, index int, ok bool) {
	index = IndexBy(list, predicate)
	if index != -1 {
		return list[index], index, true
	}
	return
}

// FindOrElse find predicted element, or return fallback if not found.
func FindOrElse[E any](list []E, fallback E, predicate func(E) bool) E {
	if elem, ok := Find(list, predicate); ok {
		return elem
	}
	return fallback
}

// MostBy returns most element by comparison, or zero value if list is empty,
// comparison need return true if prefer first element, otherwise return false.
func MostBy[E any](list []E, comparison func(E, E) bool) (elem E) {
	if len(list) == 0 {
		return
	}

	elem = list[0]
	for i := 1; i < len(list); i++ {
		if comparison(list[i], elem) {
			elem = list[i]
		}
	}
	return
}

// Min returns minimal element, or 0 if list is empty.
func Min[E constraints.Ordered](list []E) E {
	if len(list) == 0 {
		var zero E
		return zero
	}
	return slices.Min(list)
}

// Max returns maximal element, or 0 if list is empty.
func Max[E constraints.Ordered](list []E) E {
	if len(list) == 0 {
		var zero E
		return zero
	}
	return slices.Max(list)
}
