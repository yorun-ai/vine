package jsonschema

import (
	"encoding/json/v2"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
)

func scalarType(kind descriptor.Scalar) *descriptor.Type {
	return new(descriptor.Type{Kind: descriptor.TypeKindScalar, Scalar: kind})
}

func dataType(name string, args ...*descriptor.Type) *descriptor.Type {
	return new(descriptor.Type{Kind: descriptor.TypeKindData, SkelName: name, TypeArguments: args})
}

func parameter(name string) *descriptor.Type {
	return new(descriptor.Type{Kind: descriptor.TypeKindTypeParameter, Name: name})
}

func TestMethodResolvesGenericCyclesWithoutMutatingDescriptors(t *testing.T) {
	optional := parameter("T")
	optional.Nullable = true
	data := map[string]*descriptor.Data{
		"demo.Node": {
			SkelName: "demo.Node", TypeParameters: []string{"T"}, Description: "A linked node",
			Members: []*descriptor.Member{
				{Name: "value", Type: parameter("T"), Description: "Value", Sensitive: true},
				{Name: "optional", Type: optional},
				{Name: "children", Type: new(descriptor.Type{Kind: descriptor.TypeKindList, Element: dataType("demo.Node", parameter("T"))})},
			},
		},
	}
	nullableString := scalarType(descriptor.ScalarString)
	nullableString.Nullable = true
	method := new(descriptor.Method{
		Arguments: []*descriptor.Member{
			{Name: "text", Type: dataType("demo.Node", scalarType(descriptor.ScalarString))},
			{Name: "number", Type: dataType("demo.Node", scalarType(descriptor.ScalarInt))},
			{Name: "nullable", Type: dataType("demo.Node", nullableString)},
		},
		ResultType: dataType("demo.Node", scalarType(descriptor.ScalarInt)),
	})
	before, err := json.Marshal(data)
	require.NoError(t, err)
	input, output, err := Method(method, data, nil)
	require.NoError(t, err)
	require.Equal(t, dialect, input["$schema"])
	defs := input["$defs"].(map[string]Schema)
	require.Len(t, defs, 3)
	text := defs["demo.Node<string>"]
	require.Equal(t, []string{"value", "children"}, text["required"])
	props := text["properties"].(map[string]Schema)
	require.Equal(t, "string", props["value"]["type"])
	require.Equal(t, true, props["value"]["x-skel-sensitive"])
	require.Equal(t, "null", props["optional"]["anyOf"].([]Schema)[1]["type"])
	ref := props["children"]["items"].(Schema)["$ref"].(string)
	decoded, err := url.PathUnescape(strings.TrimPrefix(ref, "#/$defs/"))
	require.NoError(t, err)
	require.Equal(t, "demo.Node<string>", decoded)
	require.Equal(t, []string{"children"}, defs["demo.Node<string?>"]["required"])
	require.Len(t, output["$defs"], 1, "each schema carries only its own dependencies")
	after, err := json.Marshal(data)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
}

func TestMethodNestedGenericsMapsAndAnnotations(t *testing.T) {
	data := map[string]*descriptor.Data{
		"demo.Box": {SkelName: "demo.Box", TypeParameters: []string{"T"}, Members: []*descriptor.Member{{Name: "item", Type: parameter("T")}}},
	}
	enums := map[string]*descriptor.Enum{
		"demo.Status": {SkelName: "demo.Status", Items: []*descriptor.EnumItem{{Name: "ACTIVE", Description: "Active", Deprecated: true, DeprecatedReason: "Use READY"}}},
	}
	method := new(descriptor.Method{
		Arguments: []*descriptor.Member{{Name: "boxes", Type: new(descriptor.Type{
			Kind: descriptor.TypeKindMap, Key: new(descriptor.Type{Kind: descriptor.TypeKindEnum, SkelName: "demo.Status"}),
			Value: dataType("demo.Box", dataType("demo.Box", scalarType(descriptor.ScalarInt))),
		})}},
		InputDescription: "Find boxes", ResultType: scalarType(descriptor.ScalarDecimal), OutputExample: `"1.50"`,
	})
	input, output, err := Method(method, data, enums)
	require.NoError(t, err)
	require.Equal(t, "Find boxes", input["description"])
	defs := input["$defs"].(map[string]Schema)
	require.Len(t, defs, 3)
	require.Equal(t, "integer", defs["demo.Box<int>"]["properties"].(map[string]Schema)["item"]["type"])
	status := defs["demo.Status"]["oneOf"].([]Schema)
	require.Equal(t, "UNSPECIFIED", status[0]["const"])
	require.Equal(t, "Use READY", status[1]["x-skel-deprecatedReason"])
	require.Equal(t, true, status[1]["deprecated"])
	require.Equal(t, Schema{"$ref": "#/$defs/demo.Status"}, input["properties"].(map[string]Schema)["boxes"]["propertyNames"])
	wire, err := json.Marshal(output)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"examples":["1.50"]`)
	method.OutputExample = "1.50"
	_, output, err = Method(method, data, enums)
	require.NoError(t, err)
	require.Equal(t, []any{"1.50"}, output["examples"])
	input, output, err = Method(new(descriptor.Method), nil, nil)
	require.NoError(t, err)
	require.Equal(t, "object", input["type"])
	require.Empty(t, input["properties"])
	require.NotContains(t, input, "required")
	require.Equal(t, "null", output["type"])
}

func TestMethodRejectsIncompleteOrUnboundedSchemas(t *testing.T) {
	for _, kind := range []*descriptor.Type{
		dataType("demo.Missing"),
		new(descriptor.Type{Kind: descriptor.TypeKindEnum, SkelName: "demo.Missing"}),
		parameter("T"),
		new(descriptor.Type{Kind: "future"}),
		new(descriptor.Type{Kind: descriptor.TypeKindList}),
		scalarType("future"),
	} {
		input, output, err := Method(new(descriptor.Method{ResultType: kind}), nil, nil)
		require.Error(t, err)
		require.Nil(t, input)
		require.Nil(t, output)
	}
	data := map[string]*descriptor.Data{
		"demo.Growing": {SkelName: "demo.Growing", TypeParameters: []string{"T"}, Members: []*descriptor.Member{
			{Name: "next", Type: dataType("demo.Growing", new(descriptor.Type{Kind: descriptor.TypeKindList, Element: parameter("T")}))},
		}},
	}
	_, _, err := Method(new(descriptor.Method{ResultType: dataType("demo.Growing")}), data, nil)
	require.ErrorContains(t, err, "needs 1 type arguments")
	_, _, err = Method(new(descriptor.Method{ResultType: dataType("demo.Growing", scalarType(descriptor.ScalarString))}), data, nil)
	require.ErrorContains(t, err, "exceeds 128 levels")
}

func TestMethodDescribesMapKeysAndNullableValues(t *testing.T) {
	value := scalarType(descriptor.ScalarString)
	value.Nullable = true
	method := new(descriptor.Method{ResultType: new(descriptor.Type{
		Kind: descriptor.TypeKindMap, Key: scalarType(descriptor.ScalarInt), Value: value,
	})})
	_, output, err := Method(method, nil, nil)
	require.NoError(t, err)
	require.Equal(t, Schema{"type": "string", "pattern": `^-?[0-9]+$`}, output["propertyNames"])
	require.Equal(t, "null", output["additionalProperties"].(Schema)["anyOf"].([]Schema)[1]["type"])
	method.ResultType.Key = scalarType(descriptor.ScalarUUID)
	_, output, err = Method(method, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "uuid", output["propertyNames"].(Schema)["format"])
	method.ResultType.Key.Nullable = true
	_, _, err = Method(method, nil, nil)
	require.ErrorContains(t, err, "invalid map key")
}
