package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type cborPayload struct {
	Name  string         `cbor:"name"`
	Count int            `cbor:"count"`
	Flags map[string]any `cbor:"flags"`
}

func TestMarshalAndUnmarshalCbor(t *testing.T) {
	payload := cborPayload{
		Name:  "vine",
		Count: 3,
		Flags: map[string]any{"enabled": true},
	}

	data, err := MarshalCbor(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	decoded, err := UnmarshalCbor[cborPayload](data)
	assert.NoError(t, err)
	assert.Equal(t, payload.Name, decoded.Name)
	assert.Equal(t, payload.Count, decoded.Count)
	assert.Equal(t, true, decoded.Flags["enabled"])
}

func TestMustMarshalAndUnmarshalCbor(t *testing.T) {
	payload := cborPayload{Name: "must", Count: 7}

	data := MustMarshalCbor(payload)
	decoded := MustUnmarshalCbor[cborPayload](data)

	assert.Equal(t, payload, *decoded)
	assert.Panics(t, func() {
		MustUnmarshalCbor[cborPayload]([]byte("not-cbor"))
	})
}
