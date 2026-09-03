package vcode

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"go.yorun.ai/vine/util/vpre"
)

// MarshalJson encodes data as JSON.
func MarshalJson(data any) ([]byte, error) {
	return MarshalJsonWithOptions(data)
}

// MarshalJsonWithOptions encodes data as JSON with additional options.
func MarshalJsonWithOptions(data any, options ...json.Options) ([]byte, error) {
	return json.Marshal(data, json.JoinOptions(options...))
}

// MarshalJsonS encodes data as a JSON string.
func MarshalJsonS(data any) (string, error) {
	jsonBytes, err := MarshalJson(data)
	return string(jsonBytes), err
}

// MustMarshalJson is like MarshalJson but panics on failure.
func MustMarshalJson(data any) []byte {
	jsonBytes, err := MarshalJson(data)
	vpre.MustNil(err)
	return jsonBytes
}

// MustMarshalJsonS is like MarshalJsonS but panics on failure.
func MustMarshalJsonS(data any) string {
	return string(MustMarshalJson(data))
}

// UnmarshalJson decodes JSON data into T.
func UnmarshalJson[T any](jsonBytes []byte) (T, error) {
	target := new(T)
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return *new(T), err
	}
	return *target, nil
}

// UnmarshalJsonS decodes a JSON string into T.
func UnmarshalJsonS[T any](str string) (T, error) {
	return UnmarshalJson[T]([]byte(str))
}

// MustUnmarshalJson is like UnmarshalJson but panics on failure.
func MustUnmarshalJson[T any](jsonBytes []byte) T {
	target, err := UnmarshalJson[T](jsonBytes)
	vpre.MustNil(err)
	return target
}

// MustUnmarshalJsonS is like UnmarshalJsonS but panics on failure.
func MustUnmarshalJsonS[T any](jsonStr string) T {
	return MustUnmarshalJson[T]([]byte(jsonStr))
}

// CompactJson removes insignificant whitespace from valid JSON and panics on invalid input.
func CompactJson(raw []byte) []byte {
	value := jsontext.Value(raw).Clone()
	err := value.Compact(
		jsontext.AllowDuplicateNames(false),
		jsontext.AllowInvalidUTF8(false),
	)
	vpre.MustNil(err)
	return value
}

// CompactJsonS is the string form of CompactJson.
func CompactJsonS(raw string) string {
	return string(CompactJson([]byte(raw)))
}

// PrettifyJson indents valid JSON with four spaces and panics on invalid input.
func PrettifyJson(raw []byte) []byte {
	value := jsontext.Value(raw).Clone()
	err := value.Indent(
		jsontext.AllowDuplicateNames(false),
		jsontext.AllowInvalidUTF8(false),
		jsontext.WithIndent("    "),
	)
	vpre.MustNil(err)
	return value
}

// PrettifyJsonS is the string form of PrettifyJson.
func PrettifyJsonS(raw string) string {
	return string(PrettifyJson([]byte(raw)))
}
