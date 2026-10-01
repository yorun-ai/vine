package appcli

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func variableFlagTestSchemas() []*skel.DomainSchema {
	boolean := new(skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool})
	text := new(skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString})
	return []*skel.DomainSchema{{Data: []*skel.DataSchema{
		{SkelName: "app.Vars", Members: []*skel.MemberSchema{
			{Name: "enabled", Type: boolean},
			{Name: "optional", Type: new(skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool, Nullable: true})},
			{Name: "feature", Type: new(skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "app.Feature"})},
			{Name: "config", Type: new(skel.TypeSchema{Kind: skel.TypeKindConfig, SkelName: "app.Config"})},
			{Name: "toggles", Type: new(skel.TypeSchema{Kind: skel.TypeKindMap, Key: text, Value: boolean})},
			{Name: "groups", Type: new(skel.TypeSchema{Kind: skel.TypeKindList, Element: boolean})},
			{Name: "text", Type: text},
			{Name: "missingType", Type: new(skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "app.Missing"})},
		}},
		{SkelName: "app.Feature", Members: []*skel.MemberSchema{{Name: "enabled", Type: boolean}}},
	}, Configs: []*skel.ConfigSchema{
		{SkelName: "app.Config", Members: []*skel.MemberSchema{{Name: "enabled", Type: boolean}}},
	}}}
}

func TestVariableFlagSchemaResolvesBooleanPaths(t *testing.T) {
	schema := newVariableFlagSchema(variableFlagTestSchemas())
	for _, path := range []string{"enabled", "feature.enabled", "config.enabled", "toggles.anyKey", "optional"} {
		t.Run(path, func(t *testing.T) {
			kind := schema.variableType(path)
			require.NotNil(t, kind)
			require.Equal(t, skel.TypeKindScalar, kind.Kind)
			require.Equal(t, skel.ScalarBool, kind.Scalar)
		})
	}
	require.True(t, schema.variableType("optional").Nullable)
	require.Equal(t, skel.ScalarString, schema.variableType("text").Scalar)
	for _, path := range []string{"unknown", "feature.unknown", "enabled.child", "groups.item", "missingType.enabled"} {
		require.Nil(t, schema.variableType(path))
	}
	require.Nil(t, newVariableFlagSchema(nil).variableType("enabled"))
}
