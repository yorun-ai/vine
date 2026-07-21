package vslice

import (
	"iter"
	"slices"
)

// Collect collects values from seq into a new slice.
func Collect[E any](seq iter.Seq[E]) []E {
	return slices.Collect(seq)
}
