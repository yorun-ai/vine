package vcode

import (
	"go.yorun.ai/vine/util/vpre"
	"gopkg.in/yaml.v3"
)

// MarshalYaml encodes data as YAML.
func MarshalYaml(data any) ([]byte, error) {
	return yaml.Marshal(data)
}

// MarshalYamlS encodes data as a YAML string.
func MarshalYamlS(data any) (string, error) {
	yamlBytes, err := MarshalYaml(data)
	return string(yamlBytes), err
}

// MustMarshalYaml is like MarshalYaml but panics on failure.
func MustMarshalYaml(data any) []byte {
	yamlBytes, err := MarshalYaml(data)
	vpre.MustNil(err)
	return yamlBytes
}

// MustMarshalYamlS is like MarshalYamlS but panics on failure.
func MustMarshalYamlS(data any) string {
	return string(MustMarshalYaml(data))
}

// UnmarshalYaml decodes YAML data into T.
func UnmarshalYaml[T any](yamlBytes []byte) (T, error) {
	target := new(T)
	if err := yaml.Unmarshal(yamlBytes, target); err != nil {
		return *new(T), err
	}
	return *target, nil
}

// UnmarshalYamlS decodes a YAML string into T.
func UnmarshalYamlS[T any](yamlStr string) (T, error) {
	return UnmarshalYaml[T]([]byte(yamlStr))
}

// MustUnmarshalYaml is like UnmarshalYaml but panics on failure.
func MustUnmarshalYaml[T any](yamlBytes []byte) T {
	target, err := UnmarshalYaml[T](yamlBytes)
	vpre.MustNil(err)
	return target
}

// MustUnmarshalYamlS is like UnmarshalYamlS but panics on failure.
func MustUnmarshalYamlS[T any](yamlStr string) T {
	return MustUnmarshalYaml[T]([]byte(yamlStr))
}
