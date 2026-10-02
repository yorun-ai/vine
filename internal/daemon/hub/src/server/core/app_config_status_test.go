package core

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestAppConfigValueMatchesEnumMapKeysAndValues(t *testing.T) {
	keyType := new(skel.TypeSchema{Kind: skel.TypeKindEnum, SkelName: "demo.Region"})
	valueType := new(skel.TypeSchema{Kind: skel.TypeKindEnum, SkelName: "demo.Status"})
	enums := []*skel.EnumSchema{
		{SkelName: "demo.Region", Items: []*skel.EnumItemSchema{{Name: "EAST", Description: "East"}, {Name: "WEST"}}},
		{SkelName: "demo.Status", Items: []*skel.EnumItemSchema{{Name: "ACTIVE", Description: "Active"}, {Name: "LOCKED"}}},
	}
	for _, key := range []*skel.TypeSchema{keyType, {Kind: skel.TypeKindScalar, Scalar: skel.ScalarString}} {
		t.Run(string(key.Kind), func(t *testing.T) {
			mapType := new(skel.TypeSchema{Kind: skel.TypeKindMap, Key: key, Value: valueType})
			if key.Kind == skel.TypeKindEnum {
				assert.False(t, decodedConfigValueMatchesType(map[string]any{"UNKNOWN": "ACTIVE"}, mapType, enums))
			}
			assert.True(t, decodedConfigValueMatchesType(map[string]any{"EAST": "ACTIVE", "WEST": "LOCKED"}, mapType, enums))
			assert.False(t, decodedConfigValueMatchesType(map[string]any{"EAST": "UNKNOWN"}, mapType, enums))
			assert.False(t, decodedConfigValueMatchesType(map[string]any{"EAST": 1}, mapType, enums))
			assert.False(t, decodedConfigValueMatchesType(map[string]any{"EAST": nil}, mapType, enums))
		})
	}
}

func TestIntegerMapKeyMatchesRuntime(t *testing.T) {
	schema := &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarInt}
	for _, key := range []string{"0", "-1", "9223372036854775807", "-9223372036854775808", "1e3", "1.0", "1_000", "0x10", "01", "9223372036854775808", "-9223372036854775809"} {
		encoded, err := json.Marshal(map[string]bool{key: true})
		require.NoError(t, err)
		var target map[int64]bool
		err = json.Unmarshal(encoded, &target)
		require.Equal(t, err == nil, jsonMapKeyMatchesType(key, schema, nil), key)
	}
}

func TestStructuredConfigStatus(t *testing.T) {
	scalar := func(kind skel.Scalar) *skel.TypeSchema {
		return &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: kind}
	}
	parameter := &skel.TypeSchema{Kind: skel.TypeKindTypeParameter, Name: "T"}
	box := &skel.DataSchema{SkelName: "demo.Box", TypeParameters: []string{"T"}, Members: []*skel.MemberSchema{{Name: "value", Type: parameter}}}
	leaf := &skel.DataSchema{SkelName: "demo.Leaf", Members: []*skel.MemberSchema{
		{Name: "bytes", Type: scalar(skel.ScalarBinary)},
		{Name: "text", Type: scalar(skel.ScalarString)},
	}}
	leafType := &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: leaf.SkelName}
	nullableLeaf := *leafType
	nullableLeaf.Nullable = true
	leaf.Members = append(leaf.Members, &skel.MemberSchema{Name: "next", Type: &nullableLeaf})
	kind := &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: box.SkelName, TypeArguments: []*skel.TypeSchema{
		{Kind: skel.TypeKindMap, Key: scalar(skel.ScalarInt), Value: &skel.TypeSchema{Kind: skel.TypeKindList, Element: leafType}},
	}}
	schema := &skel.ConfigSchema{Members: []*skel.MemberSchema{{Name: "settings", Type: kind}}}
	data := []*skel.DataSchema{box, leaf}
	valid := `{"settings":{"value":{"9223372036854775807":[{"bytes":"aG\r\nVsbG8=","text":"  hello  ","next":{"bytes":"","text":"","next":null}}]}}}`
	require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(schema, valid, nil, data))
	for _, value := range []string{
		`{"settings":{"value":{"0":[{"bytes":"aG VsbG8=","text":"hello","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"%%%","text":"hello","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","text":1,"next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","text":"","next":null,"extra":true}]}}}`,
		`{"settings":{"value":{"9223372036854775808":[]}}}`,
		`{"settings":{"value":null}}`,
	} {
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(schema, value, nil, data), value)
	}
	require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(schema, valid, nil, nil))
}

func TestConfigStatusKeepsIntegerPrecision(t *testing.T) {
	schema := &skel.ConfigSchema{Members: []*skel.MemberSchema{{Name: "value", Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarInt}}}}
	for _, value := range []string{"9223372036854775807", "-9223372036854775808", "9007199254740993"} {
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(schema, `{"value":`+value+`}`, nil, nil))
	}
	for _, value := range []string{"9223372036854775808", "-9223372036854775809", "1.5"} {
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(schema, `{"value":`+value+`}`, nil, nil))
	}
}

func decodedConfigValueMatchesType(value any, kind *skel.TypeSchema, enums []*skel.EnumSchema) bool {
	raw, err := json.Marshal(value)
	return err == nil && configTypeMatches(raw, kind, enums, nil, nil)
}

func TestNullableGenericParameterBindings(t *testing.T) {
	parameter := &skel.TypeSchema{Kind: skel.TypeKindTypeParameter, Name: "TValue"}
	optional := *parameter
	optional.Nullable = true
	box := &skel.DataSchema{SkelName: "demo.Box", TypeParameters: []string{"TValue"}, Members: []*skel.MemberSchema{
		{Name: "required", Type: parameter},
		{Name: "optional", Type: &optional},
		{Name: "items", Type: &skel.TypeSchema{Kind: skel.TypeKindList, Element: &optional}},
	}}
	for _, argumentNullable := range []bool{false, true} {
		kind := &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: box.SkelName, TypeArguments: []*skel.TypeSchema{{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBinary, Nullable: argumentNullable}}}
		schema := &skel.ConfigSchema{Members: []*skel.MemberSchema{{Name: "box", Type: kind}}}
		data := []*skel.DataSchema{box}
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(schema, `{"box":{"required":"","optional":null,"items":[null,"aGVsbG8="]}}`, nil, data))
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(schema, `{"box":{"required":"","optional":"","items":[]}}`, nil, data))
		wantNull := AppConfigStatusMismatch
		if argumentNullable {
			wantNull = AppConfigStatusNormal
		}
		require.Equal(t, wantNull, AppConfigStatusFor(schema, `{"box":{"required":null,"optional":null,"items":[]}}`, nil, data))
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(schema, `{"box":{"required":"","optional":"invalid base64","items":[]}}`, nil, data))
	}
}
