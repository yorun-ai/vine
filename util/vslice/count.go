package vslice

// Count returns the number of occurrences of elem in list.
func Count[E comparable](list []E, elem E) int {
	count := 0
	for _, item := range list {
		if item == elem {
			count++
		}
	}
	return count
}
