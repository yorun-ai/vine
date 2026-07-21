package vcode

import (
	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/util/vpre"
)

// MarshalCbor encodes data as CBOR.
func MarshalCbor(data any) ([]byte, error) {
	return cbor.Marshal(data)
}

// MustMarshalCbor is like MarshalCbor but panics on failure.
func MustMarshalCbor(data any) []byte {
	cborBytes, err := MarshalCbor(data)
	vpre.MustNil(err)
	return cborBytes
}

// UnmarshalCbor decodes CBOR data into a newly allocated T.
func UnmarshalCbor[T any](cborBytes []byte) (*T, error) {
	var target T
	targetPtr := &target
	err := cbor.Unmarshal(cborBytes, targetPtr)
	if err != nil {
		return nil, err
	}
	return targetPtr, nil
}

// MustUnmarshalCbor is like UnmarshalCbor but panics on failure.
func MustUnmarshalCbor[T any](cborBytes []byte) *T {
	target, err := UnmarshalCbor[T](cborBytes)
	vpre.MustNil(err)
	return target
}
