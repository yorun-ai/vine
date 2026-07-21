package vcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type yamlPayload struct {
	Name  string   `yaml:"name"`
	Count int      `yaml:"count"`
	Tags  []string `yaml:"tags"`
}

func TestMarshalAndUnmarshalYaml(t *testing.T) {
	payload := yamlPayload{
		Name:  "vine",
		Count: 4,
		Tags:  []string{"x", "y"},
	}

	data, err := MarshalYaml(payload)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "name: vine")
	assert.Contains(t, string(data), "count: 4")

	text, err := MarshalYamlS(payload)
	assert.NoError(t, err)
	assert.Equal(t, string(data), text)

	decoded, err := UnmarshalYaml[*yamlPayload](data)
	assert.NoError(t, err)
	assert.Equal(t, payload, *decoded)

	decodedFromString, err := UnmarshalYamlS[*yamlPayload](text)
	assert.NoError(t, err)
	assert.Equal(t, payload, *decodedFromString)
}

func TestMustYamlVariants(t *testing.T) {
	payload := yamlPayload{Name: "must", Count: 1, Tags: []string{"tag"}}

	data := MustMarshalYaml(payload)
	assert.Equal(t, string(data), MustMarshalYamlS(payload))
	assert.Equal(t, payload, *MustUnmarshalYaml[*yamlPayload](data))
	assert.Equal(t, payload, *MustUnmarshalYamlS[*yamlPayload](string(data)))

	assert.Panics(t, func() {
		MustUnmarshalYaml[*yamlPayload]([]byte(":"))
	})
	assert.Panics(t, func() {
		MustUnmarshalYamlS[*yamlPayload](":")
	})
}
