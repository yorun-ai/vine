package vcode

import (
	"testing"

	"github.com/fxamacker/cbor/v2"
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

func TestMarshalCborFormatsNilContainersAsEmpty(t *testing.T) {
	payload := struct {
		Items  []string          `cbor:"items"`
		Labels map[string]string `cbor:"labels"`
	}{}

	data, err := MarshalCbor(payload)
	assert.NoError(t, err)

	var encoded map[string]cbor.RawMessage
	assert.NoError(t, cbor.Unmarshal(data, &encoded))
	assert.Equal(t, cbor.RawMessage{0x80}, encoded["items"])
	assert.Equal(t, cbor.RawMessage{0xa0}, encoded["labels"])
}

func TestMustMarshalAndUnmarshalCbor(t *testing.T) {
	payload := cborPayload{Name: "must", Count: 7, Flags: map[string]any{}}

	data := MustMarshalCbor(payload)
	decoded := MustUnmarshalCbor[cborPayload](data)

	assert.Equal(t, payload, *decoded)
	assert.Panics(t, func() {
		MustUnmarshalCbor[cborPayload]([]byte("not-cbor"))
	})
}
