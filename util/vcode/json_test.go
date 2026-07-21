package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type jsonPayload struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Tags  []string `json:"tags"`
}

func TestMarshalAndUnmarshalJson(t *testing.T) {
	payload := jsonPayload{
		Name:  "vine",
		Count: 2,
		Tags:  []string{"a", "b"},
	}

	data, err := MarshalJson(payload)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"name":"vine","count":2,"tags":["a","b"]}`, string(data))

	text, err := MarshalJsonS(payload)
	assert.NoError(t, err)
	assert.JSONEq(t, string(data), text)

	decoded, err := UnmarshalJson[*jsonPayload](data)
	assert.NoError(t, err)
	assert.Equal(t, payload, *decoded)

	decodedFromString, err := UnmarshalJsonS[*jsonPayload](text)
	assert.NoError(t, err)
	assert.Equal(t, payload, *decodedFromString)
}

func TestMustJsonVariants(t *testing.T) {
	payload := jsonPayload{Name: "must", Count: 5, Tags: []string{"x"}}

	data := MustMarshalJson(payload)
	assert.JSONEq(t, `{"name":"must","count":5,"tags":["x"]}`, string(data))
	assert.JSONEq(t, string(data), MustMarshalJsonS(payload))
	assert.Equal(t, payload, *MustUnmarshalJson[*jsonPayload](data))
	assert.Equal(t, payload, *MustUnmarshalJsonS[*jsonPayload](string(data)))

	assert.Panics(t, func() {
		MustUnmarshalJson[*jsonPayload]([]byte("{"))
	})
	assert.Panics(t, func() {
		MustUnmarshalJsonS[*jsonPayload]("{")
	})
}

func TestCompactAndPrettifyJson(t *testing.T) {
	raw := []byte("{\n  \"name\": \"vine\",\n  \"count\": 2\n}")

	compacted := CompactJson(raw)
	assert.JSONEq(t, string(raw), string(compacted))
	assert.Equal(t, `{"name":"vine","count":2}`, string(compacted))
	assert.Equal(t, string(compacted), CompactJsonS(string(raw)))

	prettified := PrettifyJson(compacted)
	assert.Contains(t, string(prettified), "\n")
	assert.Contains(t, string(prettified), `    "name"`)
	assert.Equal(t, string(prettified), PrettifyJsonS(string(compacted)))
}
