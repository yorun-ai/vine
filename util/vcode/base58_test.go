package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomBase58(t *testing.T) {
	random := RandomBase58(64)

	assert.Len(t, random, 64)
	for _, r := range random {
		assert.Contains(t, Base58Chars, string(r))
	}

	assert.Equal(t, "", RandomBase58(0))
}
