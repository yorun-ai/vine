package vslice

import (
	"cmp"
	"math/rand"
	"slices"

	"golang.org/x/exp/constraints"
)

// Clone returns a copy of slice.
func Clone[S ~[]E, E any](s S) S {
	return slices.Clone(s)
}

// Delete removes the elements s[i:j] from s.
func Delete[S ~[]E, E any](s S, i, j int) S {
	return slices.Delete(s, i, j)
}

// Unique returns a duplicate-free version of an array, in which only the first occurrence of each element is kept.
// The order of result values is determined by the order they occur in the array.
func Unique[E comparable](list []E) []E {
	return UniqueBy(list, func(elem E) E {
		return elem
	})
}

// UniqueBy returns a duplicate-free version of an array, in which only the first occurrence of each element is kept.
// The order of result values is determined by the order they occur in the array. It accepts `iteratee` which is
// invoked for each element in array to generate the criterion by which uniqueness is computed.
func UniqueBy[E any, U comparable](list []E, iteratee func(E) U) []E {
	result := make([]E, 0, len(list))
	seen := make(map[U]struct{}, len(list))

	for _, item := range list {
		key := iteratee(item)

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, item)
	}

	return result
}

// GroupBy returns an object composed of keys generated from the results of running each element of list through iteratee.
func GroupBy[E any, U comparable](list []E, iteratee func(E) U) map[U][]E {
	result := map[U][]E{}
	for _, elem := range list {
		key := iteratee(elem)
		result[key] = append(result[key], elem)
	}
	return result
}

// Flatten returns an array a single level deep.
func Flatten[E any](lists [][]E) []E {
	if len(lists) == 0 {
		return []E{}
	}
	return slices.Concat(lists...)
}

// Repeat builds a slice with N copies of initial value.
func Repeat[E any](count int, initial E) []E {
	return slices.Repeat([]E{initial}, count)
}

// RepeatBy builds a slice with values returned by N calls of callback.
func RepeatBy[E any](count int, predicate func() E) []E {
	return RepeatByI(count, func(_ int) E {
		return predicate()
	})
}

// RepeatByI do RepeatBy with index
func RepeatByI[E any](count int, predicate func(int) E) []E {
	result := make([]E, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, predicate(i))
	}
	return result
}

// Chunk returns an array of elements splits into groups the length of size. If array can't be split evenly,
// the final chunk will be the remaining elements.
func Chunk[E any](list []E, size int) [][]E {
	if size <= 0 {
		panic("Second parameter must be greater than 0")
	}
	return slices.Collect(slices.Chunk(list, size))
}

// Shuffle returns an array of shuffled values. Uses the Fisher-Yates shuffle algorithm.
// The method returns new slice rather than modify origin.
func Shuffle[E any](list []E) []E {
	result := Clone(list)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Reverse reverses array so that the first element becomes the last, the second element becomes the second to last, and so on.
// The method returns new slice rather than modify origin.
func Reverse[E any](list []E) []E {
	result := Clone(list)
	slices.Reverse(result)
	return result
}

// SortBy returns sorted slices by comparison, comparison need return true if prefer first element in the front.
// The method returns new slice rather than modify origin.
func SortBy[E any](list []E, comparison func(E, E) bool) []E {
	result := Clone(list)
	slices.SortStableFunc(result, func(left E, right E) int {
		switch {
		case comparison(left, right):
			return -1
		case comparison(right, left):
			return 1
		default:
			return 0
		}
	})
	return result
}

// Sort returns sorted ascending slice
// The method returns new slice rather than modify origin.
func Sort[E constraints.Ordered](list []E) []E {
	result := Clone(list)
	slices.Sort(result)
	return result
}

// SortDesc returns sorted descending slice
// The method returns new slice rather than modify origin.
func SortDesc[E constraints.Ordered](list []E) []E {
	result := Clone(list)
	slices.SortFunc(result, func(left E, right E) int {
		return cmp.Compare(right, left)
	})
	return result
}
