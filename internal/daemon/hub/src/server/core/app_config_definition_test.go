package core

import (
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"testing"
)

func TestConfigDefinitionPreservesReachableTypesAndMetadata(t *testing.T) {
	enum := &skel.TypeSchema{Kind: skel.TypeKindEnum, SkelName: "foreign.Mode"}
	dataType := &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "demo.Box", TypeArguments: []*skel.TypeSchema{enum}}
	recursive := &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "demo.Node", Nullable: true}
	schemas := []*skel.DataSchema{
		{Name: "Box", SkelName: "demo.Box", TypeParameters: []string{"T"}, Sensitive: true, Members: []*skel.MemberSchema{
			{Name: "value", Description: "Payload", Example: "ACTIVE", Deprecated: true, DeprecatedReason: "Old", Sensitive: true, Type: &skel.TypeSchema{Kind: skel.TypeKindTypeParameter, Name: "T"}},
			{Name: "next", Type: recursive},
		}},
		{Name: "Node", SkelName: "demo.Node", Members: []*skel.MemberSchema{{Name: "next", Type: recursive}}},
		{SkelName: "demo.Unrelated"},
	}
	definition := NewAppConfigDefinition(&skel.ConfigSchema{Sensitive: true, Members: []*skel.MemberSchema{{Name: "box", Type: dataType}}},
		[]*skel.EnumSchema{{SkelName: "foreign.Mode", Items: []*skel.EnumItemSchema{{Name: "ACTIVE", Description: "Active"}}}}, schemas)
	require.True(t, definition.Sensitive)
	require.Equal(t, "demo.Box<foreign.Mode>", definition.Fields[0].Type)
	require.Equal(t, "Active", definition.Fields[0].ValueType.TypeArguments[0].EnumItems[0].Description)
	require.Len(t, definition.DataTypes, 2)
	box := definition.DataTypes[0]
	require.True(t, box.Sensitive)
	require.Equal(t, []string{"T"}, box.TypeParameters)
	require.True(t, box.Fields[0].Sensitive)
	require.True(t, box.Fields[0].Deprecated)
	require.Equal(t, "ACTIVE", box.Fields[0].Example)
	require.Equal(t, "typeParameter", box.Fields[0].ValueType.Kind)
}
