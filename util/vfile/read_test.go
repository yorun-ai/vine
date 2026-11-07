package vfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type readPayload struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestReadAsBytesAndString(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.txt")

	assert.NoError(t, WriteString(path, "hello vine"))

	contentBytes, err := ReadAsBytes(path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello vine"), contentBytes)
	assert.Equal(t, []byte("hello vine"), MustReadAsBytes(path))

	contentString, err := ReadAsString(path)
	assert.NoError(t, err)
	assert.Equal(t, "hello vine", contentString)
	assert.Equal(t, "hello vine", MustReadAsString(path))
}

func TestReadAsBytesMissingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "missing.txt")

	_, err := ReadAsBytes(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not exist")

	assert.Panics(t, func() {
		MustReadAsBytes(path)
	})
	assert.Panics(t, func() {
		MustReadAsString(path)
	})
}

func TestReadAsJsonAndYaml(t *testing.T) {
	root := t.TempDir()
	jsonPath := filepath.Join(root, "payload.json")
	yamlPath := filepath.Join(root, "payload.yaml")

	assert.NoError(t, WriteString(jsonPath, `{"name":"vine","count":5}`))
	assert.NoError(t, WriteString(yamlPath, "name: vine\ncount: 6\n"))

	jsonPayload, err := ReadAsJson[*readPayload](jsonPath)
	assert.NoError(t, err)
	assert.Equal(t, readPayload{Name: "vine", Count: 5}, *jsonPayload)
	assert.Equal(t, readPayload{Name: "vine", Count: 5}, *MustReadAsJson[*readPayload](jsonPath))

	yamlPayload, err := ReadAsYaml[*readPayload](yamlPath)
	assert.NoError(t, err)
	assert.Equal(t, readPayload{Name: "vine", Count: 6}, *yamlPayload)
	assert.Equal(t, readPayload{Name: "vine", Count: 6}, *MustReadAsYaml[*readPayload](yamlPath))
}

func TestReadAsJsonAndYamlInvalidContent(t *testing.T) {
	root := t.TempDir()
	jsonPath := filepath.Join(root, "invalid.json")
	yamlPath := filepath.Join(root, "invalid.yaml")

	assert.NoError(t, WriteString(jsonPath, "{"))
	assert.NoError(t, WriteString(yamlPath, ":"))

	_, err := ReadAsJson[*readPayload](jsonPath)
	assert.Error(t, err)
	_, err = ReadAsYaml[*readPayload](yamlPath)
	assert.Error(t, err)

	assert.Panics(t, func() {
		MustReadAsJson[*readPayload](jsonPath)
	})
	assert.Panics(t, func() {
		MustReadAsYaml[*readPayload](yamlPath)
	})
}
