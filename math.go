package dsa

import (
	"math/bits"

	"golang.org/x/exp/constraints"
)

// Log2 calculates the base-2 logarithm of a given positive integer `x`.
// If x is not greater than 0, it returns 0.
func Log2[I constraints.Integer](x I) int {
	if x <= 0 {
		return 0
	}
	return bits.Len64(uint64(x)) - 1
}
