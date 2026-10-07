package core

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
)

func TestAppConfigValueMatchesEnumMapKeysAndValues(t *testing.T) {
	keyType := new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "demo.Region"})
	valueType := new(skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "demo.Status"})
	enums := []*skeldesc.Enum{
		{SkelName: "demo.Region", Items: []*skeldesc.EnumItem{{Name: "EAST", Description: "East"}, {Name: "WEST"}}},
		{SkelName: "demo.Status", Items: []*skeldesc.EnumItem{{Name: "ACTIVE", Description: "Active"}, {Name: "LOCKED"}}},
	}
	for _, key := range []*skeldesc.Type{keyType, {Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString}} {
		t.Run(string(key.Kind), func(t *testing.T) {
			mapType := new(skeldesc.Type{Kind: skeldesc.TypeKindMap, Key: key, Value: valueType})
			if key.Kind == skeldesc.TypeKindEnum {
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
	descriptor := &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt}
	for _, key := range []string{"0", "-1", "9223372036854775807", "-9223372036854775808", "1e3", "1.0", "1_000", "0x10", "01", "9223372036854775808", "-9223372036854775809"} {
		encoded, err := json.Marshal(map[string]bool{key: true})
		require.NoError(t, err)
		var target map[int64]bool
		err = json.Unmarshal(encoded, &target)
		require.Equal(t, err == nil, jsonMapKeyMatchesType(key, descriptor, nil), key)
	}
}

func TestStructuredConfigStatus(t *testing.T) {
	scalar := func(kind skeldesc.Scalar) *skeldesc.Type {
		return &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: kind}
	}
	parameter := &skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "T"}
	box := &skeldesc.Data{SkelName: "demo.Box", TypeParameters: []string{"T"}, Members: []*skeldesc.Member{{Name: "value", Type: parameter}}}
	leaf := &skeldesc.Data{SkelName: "demo.Leaf", Members: []*skeldesc.Member{
		{Name: "bytes", Type: scalar(skeldesc.ScalarBinary)},
		{Name: "text", Type: scalar(skeldesc.ScalarString)},
	}}
	leafType := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: leaf.SkelName}
	nullableLeaf := *leafType
	nullableLeaf.Nullable = true
	leaf.Members = append(leaf.Members, &skeldesc.Member{Name: "next", Type: &nullableLeaf})
	kind := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: box.SkelName, TypeArguments: []*skeldesc.Type{
		{Kind: skeldesc.TypeKindMap, Key: scalar(skeldesc.ScalarInt), Value: &skeldesc.Type{Kind: skeldesc.TypeKindList, Element: leafType}},
	}}
	descriptor := &skeldesc.Config{Members: []*skeldesc.Member{{Name: "settings", Type: kind}}, Lifecycle: skeldesc.ConfigLifecycleEternal}
	data := []*skeldesc.Data{box, leaf}
	valid := `{"settings":{"value":{"9223372036854775807":[{"bytes":"aG\r\nVsbG8=","text":"  hello  ","next":{"bytes":"","text":"","next":null}}]}}}`
	require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(descriptor, valid, nil, data))
	for _, value := range []string{
		`{"settings":{"value":{"0":[{"bytes":"aG VsbG8=","text":"hello","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"%%%","text":"hello","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","text":1,"next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","next":null}]}}}`,
		`{"settings":{"value":{"0":[{"bytes":"","text":"","next":null,"extra":true}]}}}`,
		`{"settings":{"value":{"9223372036854775808":[]}}}`,
		`{"settings":{"value":null}}`,
	} {
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(descriptor, value, nil, data), value)
	}
	require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(descriptor, valid, nil, nil))
}

func TestConfigStatusKeepsIntegerPrecision(t *testing.T) {
	descriptor := &skeldesc.Config{Members: []*skeldesc.Member{{Name: "value", Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt}}}, Lifecycle: skeldesc.ConfigLifecycleEternal}
	for _, value := range []string{"9223372036854775807", "-9223372036854775808", "9007199254740993"} {
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(descriptor, `{"value":`+value+`}`, nil, nil))
	}
	for _, value := range []string{"9223372036854775808", "-9223372036854775809", "1.5"} {
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(descriptor, `{"value":`+value+`}`, nil, nil))
	}
}

func decodedConfigValueMatchesType(value any, kind *skeldesc.Type, enums []*skeldesc.Enum) bool {
	raw, err := json.Marshal(value)
	return err == nil && configTypeMatches(raw, kind, enums, nil, nil)
}

func TestNullableGenericParameterBindings(t *testing.T) {
	parameter := &skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "TValue"}
	optional := *parameter
	optional.Nullable = true
	box := &skeldesc.Data{SkelName: "demo.Box", TypeParameters: []string{"TValue"}, Members: []*skeldesc.Member{
		{Name: "required", Type: parameter},
		{Name: "optional", Type: &optional},
		{Name: "items", Type: &skeldesc.Type{Kind: skeldesc.TypeKindList, Element: &optional}},
	}}
	for _, argumentNullable := range []bool{false, true} {
		kind := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: box.SkelName, TypeArguments: []*skeldesc.Type{{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBinary, Nullable: argumentNullable}}}
		descriptor := &skeldesc.Config{Members: []*skeldesc.Member{{Name: "box", Type: kind}}, Lifecycle: skeldesc.ConfigLifecycleEternal}
		data := []*skeldesc.Data{box}
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(descriptor, `{"box":{"required":"","optional":null,"items":[null,"aGVsbG8="]}}`, nil, data))
		require.Equal(t, AppConfigStatusNormal, AppConfigStatusFor(descriptor, `{"box":{"required":"","optional":"","items":[]}}`, nil, data))
		wantNull := AppConfigStatusMismatch
		if argumentNullable {
			wantNull = AppConfigStatusNormal
		}
		require.Equal(t, wantNull, AppConfigStatusFor(descriptor, `{"box":{"required":null,"optional":null,"items":[]}}`, nil, data))
		require.Equal(t, AppConfigStatusMismatch, AppConfigStatusFor(descriptor, `{"box":{"required":"","optional":"invalid base64","items":[]}}`, nil, data))
	}
}
