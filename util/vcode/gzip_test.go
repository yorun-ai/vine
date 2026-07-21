package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipAndUngzip(t *testing.T) {
	source := []byte("vine utilities test payload")

	zipped, err := Gzip(source)
	assert.NoError(t, err)
	assert.NotEmpty(t, zipped)

	unzipped, err := Ungzip(zipped)
	assert.NoError(t, err)
	assert.Equal(t, source, unzipped)
}

func TestGzipStringHelpers(t *testing.T) {
	source := "string payload"

	zipped, err := GzipS(source)
	assert.NoError(t, err)

	unzipped, err := UngzipS(zipped)
	assert.NoError(t, err)
	assert.Equal(t, source, unzipped)
}

func TestMustGzipAndUngzip(t *testing.T) {
	source := []byte("must payload")

	zipped := MustGzip(source)
	assert.Equal(t, source, MustUngzip(zipped))
	assert.Equal(t, "must payload", MustUngzipS(zipped))
	assert.Equal(t, zipped, MustGzipS("must payload"))

	assert.Panics(t, func() {
		MustUngzip([]byte("not-gzip"))
	})
	assert.Panics(t, func() {
		MustUngzipS([]byte("not-gzip"))
	})
}
