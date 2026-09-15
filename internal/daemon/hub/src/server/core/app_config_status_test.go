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
				assert.False(t, jsonValueMatchesType(map[string]any{"UNKNOWN": "ACTIVE"}, mapType, enums))
			}
			assert.True(t, jsonValueMatchesType(map[string]any{"EAST": "ACTIVE", "WEST": "LOCKED"}, mapType, enums))
			assert.False(t, jsonValueMatchesType(map[string]any{"EAST": "UNKNOWN"}, mapType, enums))
			assert.False(t, jsonValueMatchesType(map[string]any{"EAST": 1}, mapType, enums))
			assert.False(t, jsonValueMatchesType(map[string]any{"EAST": nil}, mapType, enums))
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
