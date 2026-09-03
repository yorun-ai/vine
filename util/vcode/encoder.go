package vcode

import (
	"encoding/json/v2"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/util/vpre"
)

// Encoder applies configured JSON and CBOR encoding behavior.
type Encoder struct {
	jsonOptions json.Options
	cborMode    cbor.EncMode
}

var defaultEncoder = newDefaultEncoder()

func newDefaultEncoder() Encoder {
	cborOptions := cbor.EncOptions{NilContainers: cbor.NilContainerAsEmpty}
	cborMode, err := cborOptions.EncMode()
	vpre.MustNil(err)
	return NewEncoder(nil, cborMode)
}

// NewEncoder creates an encoder with explicit JSON options and CBOR mode.
func NewEncoder(jsonOptions json.Options, cborMode cbor.EncMode) Encoder {
	return Encoder{jsonOptions: jsonOptions, cborMode: cborMode}
}

// DefaultEncoder returns the encoder for Vine's current wire behavior.
func DefaultEncoder() Encoder {
	return defaultEncoder
}

// MarshalJson encodes data as JSON using the encoder's collection profile.
func (e Encoder) MarshalJson(data any) ([]byte, error) {
	if e.jsonOptions == nil {
		return json.Marshal(data)
	}
	return json.Marshal(data, e.jsonOptions)
}

// MustMarshalJson is like MarshalJson but panics on failure.
func (e Encoder) MustMarshalJson(data any) []byte {
	dataBytes, err := e.MarshalJson(data)
	vpre.MustNil(err)
	return dataBytes
}

// MustMarshalJsonS is the string form of MustMarshalJson.
func (e Encoder) MustMarshalJsonS(data any) string {
	return string(e.MustMarshalJson(data))
}

// MarshalCbor encodes data as CBOR using the encoder's collection profile.
func (e Encoder) MarshalCbor(data any) ([]byte, error) {
	return e.cborMode.Marshal(data)
}

// MustMarshalCbor is like MarshalCbor but panics on failure.
func (e Encoder) MustMarshalCbor(data any) []byte {
	dataBytes, err := e.MarshalCbor(data)
	vpre.MustNil(err)
	return dataBytes
}
