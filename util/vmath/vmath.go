package vmath

import (
	"math/rand"
	"time"

	"go.yorun.ai/vine/util/vpre"
)

var rander = rand.New(rand.NewSource(time.Now().Unix()))

// Numeric is the set of numeric types supported by this package's generic helpers.
type Numeric interface {
	int | int64 | float32 | float64 | time.Duration
}

// InRange reports whether num is in the half-open interval [min, max).
func InRange[N Numeric](num, min, max N) bool {
	return num >= min && num < max
}

// RandIntBetween returns a pseudorandom integer in the inclusive interval [left, right].
// It panics when right is not greater than left.
func RandIntBetween(left, right int) int {
	vpre.Check(right > left, "invalid interval, right(%d) was less than or equal to left(%d)", right, left)
	return rander.Intn(right-left+1) + left
}
