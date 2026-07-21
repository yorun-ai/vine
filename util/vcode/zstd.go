package vcode

import (
	"github.com/klauspost/compress/zstd"

	"go.yorun.ai/vine/util/vpre"
)

// Zstd compresses src as a Zstandard frame.
func Zstd(src []byte) ([]byte, error) {
	var dst []byte
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(src, dst), nil
}

// ZstdS compresses the UTF-8 bytes of src as a Zstandard frame.
func ZstdS(src string) ([]byte, error) {
	return Zstd([]byte(src))
}

// MustZstd is like Zstd but panics on failure.
func MustZstd(src []byte) []byte {
	zipped, err := Zstd(src)
	vpre.CheckNilError(err, "Zstd error")
	return zipped
}

// MustZstdS is like ZstdS but panics on failure.
func MustZstdS(src string) []byte {
	return MustZstd([]byte(src))
}

// Unzstd decompresses a Zstandard frame.
func Unzstd(zipped []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return decoder.DecodeAll(zipped, nil)
}

// UnzstdS decompresses a Zstandard frame and returns it as a string.
func UnzstdS(zipped []byte) (string, error) {
	unzipped, err := Unzstd(zipped)
	return string(unzipped), err
}

// MustUnzstd is like Unzstd but panics on failure.
func MustUnzstd(zipped []byte) []byte {
	unzipped, err := Unzstd(zipped)
	vpre.CheckNilError(err, "Unzstd error")
	return unzipped
}

// MustUnzstdS is like UnzstdS but panics on failure.
func MustUnzstdS(zipped []byte) string {
	return string(MustUnzstd(zipped))
}
