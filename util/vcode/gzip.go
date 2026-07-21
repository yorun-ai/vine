package vcode

import (
	"bytes"
	"compress/gzip"
	"io"

	"go.yorun.ai/vine/util/vpre"
)

// Gzip compresses src in gzip format.
func Gzip(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(src); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GzipS compresses the UTF-8 bytes of src in gzip format.
func GzipS(src string) ([]byte, error) {
	return Gzip([]byte(src))
}

// MustGzip is like Gzip but panics on failure.
func MustGzip(src []byte) []byte {
	zipped, err := Gzip(src)
	vpre.CheckNilError(err, "Gzip error")
	return zipped
}

// MustGzipS is like GzipS but panics on failure.
func MustGzipS(src string) []byte {
	return MustGzip([]byte(src))
}

// Ungzip decompresses gzip data.
func Ungzip(zipped []byte) ([]byte, error) {
	reader := bytes.NewReader(zipped)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	unzipped, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}
	return unzipped, nil
}

// UngzipS decompresses gzip data and returns it as a string.
func UngzipS(zipped []byte) (string, error) {
	unzipped, err := Ungzip(zipped)
	return string(unzipped), err
}

// MustUngzip is like Ungzip but panics on failure.
func MustUngzip(zipped []byte) []byte {
	unzipped, err := Ungzip(zipped)
	vpre.CheckNilError(err, "Ungzip error")
	return unzipped
}

// MustUngzipS is like UngzipS but panics on failure.
func MustUngzipS(zipped []byte) string {
	return string(MustUngzip(zipped))
}
