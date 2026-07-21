package vfile

import (
	"fmt"
	"os"

	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

// ReadAsBytes reads the file at path.
func ReadAsBytes(path string) ([]byte, error) {
	if !ExistsFile(path) {
		return nil, fmt.Errorf("%s not exist", path)
	}
	return os.ReadFile(path)
}

// MustReadAsBytes is like ReadAsBytes but panics on failure.
func MustReadAsBytes(path string) []byte {
	contentBytes, err := ReadAsBytes(path)
	vpre.MustNil(err)
	return contentBytes
}

// ReadAsString reads the file at path as a string.
func ReadAsString(path string) (string, error) {
	contentBytes, err := ReadAsBytes(path)
	if err != nil {
		return "", err
	}
	return string(contentBytes), nil
}

// MustReadAsString is like ReadAsString but panics on failure.
func MustReadAsString(path string) string {
	contentStr, err := ReadAsString(path)
	vpre.MustNil(err)
	return contentStr
}

// ReadAsJson reads path and decodes its JSON content into T.
func ReadAsJson[T any](path string) (T, error) {
	contentBytes, err := ReadAsBytes(path)
	if err != nil {
		return *new(T), err
	}
	return vcode.UnmarshalJson[T](contentBytes)
}

// MustReadAsJson is like ReadAsJson but panics on failure.
func MustReadAsJson[T any](path string) T {
	unmarshalled, err := ReadAsJson[T](path)
	vpre.MustNil(err)
	return unmarshalled
}

// ReadAsYaml reads path and decodes its YAML content into T.
func ReadAsYaml[T any](path string) (T, error) {
	contentBytes, err := ReadAsBytes(path)
	if err != nil {
		return *new(T), err
	}
	return vcode.UnmarshalYaml[T](contentBytes)
}

// MustReadAsYaml is like ReadAsYaml but panics on failure.
func MustReadAsYaml[T any](path string) T {
	unmarshalled, err := ReadAsYaml[T](path)
	vpre.MustNil(err)
	return unmarshalled
}
