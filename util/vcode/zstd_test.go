package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZstdAndUnzstd(t *testing.T) {
	source := []byte("vine utilities test payload")

	zipped, err := Zstd(source)
	assert.NoError(t, err)
	assert.NotEmpty(t, zipped)

	unzipped, err := Unzstd(zipped)
	assert.NoError(t, err)
	assert.Equal(t, source, unzipped)
}

func TestZstdStringHelpers(t *testing.T) {
	source := "string payload"

	zipped, err := ZstdS(source)
	assert.NoError(t, err)

	unzipped, err := UnzstdS(zipped)
	assert.NoError(t, err)
	assert.Equal(t, source, unzipped)
}

func TestMustZstdAndUnzstd(t *testing.T) {
	source := []byte("must payload")

	zipped := MustZstd(source)
	assert.Equal(t, source, MustUnzstd(zipped))
	assert.Equal(t, "must payload", MustUnzstdS(zipped))
	assert.Equal(t, zipped, MustZstdS("must payload"))

	assert.Panics(t, func() {
		MustUnzstd([]byte("not-zstd"))
	})
	assert.Panics(t, func() {
		MustUnzstdS([]byte("not-zstd"))
	})
}
