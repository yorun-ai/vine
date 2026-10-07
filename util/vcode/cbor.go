package vcode

import (
	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/util/vpre"
)

var cborEncodeMode cbor.EncMode
var cborDecodeMode cbor.DecMode

func init() {
	var err error
	cborEncodeMode, err = (cbor.EncOptions{
		NilContainers: cbor.NilContainerAsEmpty,
	}).EncMode()
	vpre.MustNil(err)

	cborDecodeMode, err = (cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}).DecMode()
	vpre.MustNil(err)
}

// MarshalCbor encodes data as CBOR, representing nil collections as empty collections.
func MarshalCbor(data any) ([]byte, error) {
	return cborEncodeMode.Marshal(data)
}

// MustMarshalCbor is like MarshalCbor but panics on failure.
func MustMarshalCbor(data any) []byte {
	dataBytes, err := MarshalCbor(data)
	vpre.MustNil(err)
	return dataBytes
}

// UnmarshalCbor decodes CBOR data into a newly allocated T, rejecting duplicate map keys.
func UnmarshalCbor[T any](cborBytes []byte) (*T, error) {
	target := new(T)
	if err := cborDecodeMode.Unmarshal(cborBytes, target); err != nil {
		return nil, err
	}
	return target, nil
}

// MustUnmarshalCbor is like UnmarshalCbor but panics on failure.
func MustUnmarshalCbor[T any](cborBytes []byte) *T {
	target, err := UnmarshalCbor[T](cborBytes)
	vpre.MustNil(err)
	return target
}
