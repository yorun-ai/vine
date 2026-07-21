package vslice

import "slices"

// Concat combine all lists into one list.
func Concat[E any](lists ...[]E) []E {
	if len(lists) == 0 {
		return []E{}
	}
	return slices.Concat(lists...)
}

// Intersect returns the intersection between two lists.
func Intersect[E comparable](list1 []E, list2 []E) []E {
	result := []E{}
	seen := map[E]struct{}{}

	for _, elem := range list1 {
		seen[elem] = struct{}{}
	}

	for _, elem := range list2 {
		if _, ok := seen[elem]; ok {
			result = append(result, elem)
		}
	}

	return result
}

// Union returns all distinct elements from both lists.
// result returns will not change the order of elements relatively.
func Union[E comparable](list1 []E, list2 []E) []E {
	result := []E{}

	seen := map[E]struct{}{}
	hasAdd := map[E]struct{}{}

	for _, e := range list1 {
		seen[e] = struct{}{}
	}

	for _, e := range list2 {
		seen[e] = struct{}{}
	}

	for _, e := range list1 {
		if _, ok := seen[e]; ok {
			result = append(result, e)
			hasAdd[e] = struct{}{}
		}
	}

	for _, e := range list2 {
		if _, ok := hasAdd[e]; ok {
			continue
		}
		if _, ok := seen[e]; ok {
			result = append(result, e)
		}
	}

	return result
}

// Difference returns the difference between two lists.
// The first value is the list of element absent of list2.
// The second value is the list of element absent of list1.
func Difference[E comparable](list1 []E, list2 []E) ([]E, []E) {
	left := []E{}
	right := []E{}

	seenLeft := map[E]struct{}{}
	seenRight := map[E]struct{}{}

	for _, elem := range list1 {
		seenLeft[elem] = struct{}{}
	}

	for _, elem := range list2 {
		seenRight[elem] = struct{}{}
	}

	for _, elem := range list1 {
		if _, ok := seenRight[elem]; !ok {
			left = append(left, elem)
		}
	}

	for _, elem := range list2 {
		if _, ok := seenLeft[elem]; !ok {
			right = append(right, elem)
		}
	}

	return left, right
}

// Equal returns true if all elements in two lists are same
func Equal[E comparable](list1 []E, list2 []E) bool {
	return slices.Equal(list1, list2)
}

// EqualDisorderly returns true if all elements in two lists are same, without order check
func EqualDisorderly[E comparable](list1 []E, list2 []E) bool {
	left, right := Difference(list1, list2)
	return len(left) == 0 && len(right) == 0
}
