package core

import (
	"testing"

	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
)

func TestConfigDefinitionPreservesReachableTypesAndMetadata(t *testing.T) {
	enum := &skeldesc.Type{Kind: skeldesc.TypeKindEnum, SkelName: "foreign.Mode"}
	dataType := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.Box", TypeArguments: []*skeldesc.Type{enum}}
	recursive := &skeldesc.Type{Kind: skeldesc.TypeKindData, SkelName: "demo.Node", Nullable: true}
	descriptors := []*skeldesc.Data{
		{Name: "Box", SkelName: "demo.Box", TypeParameters: []string{"T"}, Sensitive: true, Members: []*skeldesc.Member{
			{Name: "value", Description: "Payload", Example: "ACTIVE", Deprecated: true, DeprecatedReason: "Old", Sensitive: true, Type: &skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "T"}},
			{Name: "next", Type: recursive},
		}},
		{Name: "Node", SkelName: "demo.Node", Members: []*skeldesc.Member{{Name: "next", Type: recursive}}},
		{SkelName: "demo.Unrelated"},
	}
	definition := NewAppConfigDefinition(&skeldesc.Config{Sensitive: true, Members: []*skeldesc.Member{{Name: "box", Type: dataType}}, Lifecycle: skeldesc.ConfigLifecycleEternal},
		[]*skeldesc.Enum{{SkelName: "foreign.Mode", Items: []*skeldesc.EnumItem{{Name: "ACTIVE", Description: "Active"}}}}, descriptors)
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
