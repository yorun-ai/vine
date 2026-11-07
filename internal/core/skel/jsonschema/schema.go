// Package jsonschema describes Skel's JSON wire values using JSON Schema 2020-12.
package jsonschema

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/url"
	"strings"

	"go.yorun.ai/skel/descriptor"
)

const dialect = "https://json-schema.org/draft/2020-12/schema"

// Schema is a JSON Schema object. Each method schema is a self-contained document.
type Schema map[string]any

type _Builder struct {
	data  map[string]*descriptor.Data
	enums map[string]*descriptor.Enum
	defs  map[string]Schema
}

// Method returns schemas for the named params object and the result value,
// excluding the vRPC envelope. The supplied descriptors are never modified.
func Method(method *descriptor.Method, data map[string]*descriptor.Data, enums map[string]*descriptor.Enum) (Schema, Schema, error) {
	input := &_Builder{data: data, enums: enums, defs: map[string]Schema{}}
	params, err := input.members(method.Arguments, nil, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("input: %w", err)
	}
	annotate(params, method.InputDescription, false, "", method.ArgumentsSensitive, "")
	input.document(params)

	output := &_Builder{data: data, enums: enums, defs: map[string]Schema{}}
	result := Schema{"type": "null"}
	if method.ResultType != nil {
		result, err = output.schema(method.ResultType, nil, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("output: %w", err)
		}
	}
	annotate(result, method.OutputDescription, false, "", method.ResultSensitive, method.OutputExample)
	output.document(result)
	return params, result, nil
}

func (b *_Builder) document(schema Schema) {
	schema["$schema"] = dialect
	if len(b.defs) > 0 {
		schema["$defs"] = b.defs
	}
}

func (b *_Builder) members(members []*descriptor.Member, bindings map[string]*descriptor.Type, depth int) (Schema, error) {
	properties := map[string]Schema{}
	required := []string{}
	for _, member := range members {
		kind, err := substitute(member.Type, bindings, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", member.Name, err)
		}
		schema, err := b.schema(kind, nil, depth+1)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", member.Name, err)
		}
		annotate(schema, member.Description, member.Deprecated, member.DeprecatedReason, member.Sensitive, member.Example)
		properties[member.Name] = schema
		if !kind.Nullable {
			required = append(required, member.Name)
		}
	}
	result := Schema{"type": "object", "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	// The wire decoders accept unknown members for forward compatibility.
	return result, nil
}

func (b *_Builder) schema(kind *descriptor.Type, bindings map[string]*descriptor.Type, depth int) (Schema, error) {
	// Regular recursive declarations terminate through $ref. Expanding generic
	// recursion (e.g. Node<T> -> Node<List<T>>) has no finite schema graph.
	if depth > 128 {
		return nil, fmt.Errorf("type expansion exceeds 128 levels")
	}
	kind, err := substitute(kind, bindings, 0)
	if err != nil {
		return nil, err
	}
	var result Schema
	switch kind.Kind {
	case descriptor.TypeKindScalar:
		result, err = scalar(kind.Scalar)
	case descriptor.TypeKindList:
		var item Schema
		item, err = b.schema(kind.Element, nil, depth+1)
		result = Schema{"type": "array", "items": item}
	case descriptor.TypeKindMap:
		var value Schema
		value, err = b.schema(kind.Value, nil, depth+1)
		if err != nil {
			break
		}
		var key Schema
		key, err = b.mapKey(kind.Key, depth+1)
		result = Schema{"type": "object", "additionalProperties": value, "propertyNames": key}
	case descriptor.TypeKindData, descriptor.TypeKindEnum:
		result, err = b.named(kind, depth+1)
	default:
		err = fmt.Errorf("unsupported type kind %q", kind.Kind)
	}
	if err != nil {
		return nil, err
	}
	if kind.Nullable {
		result = Schema{"anyOf": []Schema{result, {"type": "null"}}}
	}
	return result, nil
}

func (b *_Builder) named(kind *descriptor.Type, depth int) (Schema, error) {
	concrete := *kind
	concrete.Nullable = false
	key := typeName(&concrete)
	ref := Schema{"$ref": "#/$defs/" + url.PathEscape(strings.NewReplacer("~", "~0", "/", "~1").Replace(key))}
	if _, exists := b.defs[key]; exists {
		return ref, nil
	}
	b.defs[key] = nil // Reserve the definition before following recursive members.
	if kind.Kind == descriptor.TypeKindEnum {
		enum, ok := b.enums[kind.SkelName]
		if !ok {
			return nil, fmt.Errorf("missing enum %s", kind.SkelName)
		}
		// Both generated Go and TypeScript enums include the implicit fallback.
		items := []Schema{{"const": "UNSPECIFIED"}}
		for _, item := range enum.Items {
			value := Schema{"const": item.Name}
			annotate(value, item.Description, item.Deprecated, item.DeprecatedReason, false, "")
			if item.Name == "UNSPECIFIED" {
				items[0] = value
			} else {
				items = append(items, value)
			}
		}
		definition := Schema{"type": "string", "oneOf": items, "title": enum.SkelName}
		annotate(definition, enum.Description, enum.Deprecated, enum.DeprecatedReason, false, "")
		b.defs[key] = definition
		return ref, nil
	}
	data, ok := b.data[kind.SkelName]
	if !ok {
		return nil, fmt.Errorf("missing data %s", kind.SkelName)
	}
	if len(data.TypeParameters) != len(kind.TypeArguments) {
		return nil, fmt.Errorf("%s needs %d type arguments, got %d", kind.SkelName, len(data.TypeParameters), len(kind.TypeArguments))
	}
	bindings := map[string]*descriptor.Type{}
	for i, name := range data.TypeParameters {
		bindings[name] = kind.TypeArguments[i]
	}
	definition, err := b.members(data.Members, bindings, depth)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	definition["title"] = key
	annotate(definition, data.Description, data.Deprecated, data.DeprecatedReason, data.Sensitive, "")
	b.defs[key] = definition
	return ref, nil
}

func (b *_Builder) mapKey(kind *descriptor.Type, depth int) (Schema, error) {
	if kind == nil || kind.Nullable {
		return nil, fmt.Errorf("invalid map key")
	}
	if kind.Kind == descriptor.TypeKindEnum {
		return b.schema(kind, nil, depth)
	}
	if kind.Kind == descriptor.TypeKindScalar {
		switch kind.Scalar {
		case descriptor.ScalarInt:
			return Schema{"type": "string", "pattern": `^-?[0-9]+$`}, nil
		case descriptor.ScalarString, descriptor.ScalarUUID:
			return scalar(kind.Scalar)
		}
	}
	return nil, fmt.Errorf("unsupported map key %s", typeName(kind))
}

// substitute produces a concrete type, retaining nullability at both the use
// site and the argument. Bindings are already concrete, including nested ones.
func substitute(kind *descriptor.Type, bindings map[string]*descriptor.Type, depth int) (*descriptor.Type, error) {
	if kind == nil {
		return nil, fmt.Errorf("missing type")
	}
	if depth > 128 {
		return nil, fmt.Errorf("type nesting exceeds 128 levels")
	}
	if kind.Kind == descriptor.TypeKindTypeParameter {
		bound, ok := bindings[kind.Name]
		if !ok {
			return nil, fmt.Errorf("unbound type parameter %s", kind.Name)
		}
		result := *bound
		result.Nullable = result.Nullable || kind.Nullable
		return &result, nil
	}
	result := *kind
	var err error
	for _, pair := range []struct {
		source *descriptor.Type
		target **descriptor.Type
	}{{kind.Element, &result.Element}, {kind.Key, &result.Key}, {kind.Value, &result.Value}} {
		if pair.source != nil {
			*pair.target, err = substitute(pair.source, bindings, depth+1)
			if err != nil {
				return nil, err
			}
		}
	}
	result.TypeArguments = make([]*descriptor.Type, len(kind.TypeArguments))
	for i, arg := range kind.TypeArguments {
		result.TypeArguments[i], err = substitute(arg, bindings, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func typeName(kind *descriptor.Type) string {
	var name string
	switch kind.Kind {
	case descriptor.TypeKindScalar:
		name = string(kind.Scalar)
	case descriptor.TypeKindList:
		name = "List<" + typeName(kind.Element) + ">"
	case descriptor.TypeKindMap:
		name = "Map<" + typeName(kind.Key) + "," + typeName(kind.Value) + ">"
	default:
		name = kind.SkelName
		if len(kind.TypeArguments) > 0 {
			args := make([]string, len(kind.TypeArguments))
			for i, arg := range kind.TypeArguments {
				args[i] = typeName(arg)
			}
			name += "<" + strings.Join(args, ",") + ">"
		}
	}
	if kind.Nullable {
		name += "?"
	}
	return name
}

func annotate(schema Schema, description string, deprecated bool, reason string, sensitive bool, example string) {
	if description != "" {
		schema["description"] = description
	}
	if deprecated {
		schema["deprecated"] = true
	}
	if reason != "" {
		schema["x-skel-deprecatedReason"] = reason
	}
	if sensitive {
		schema["x-skel-sensitive"] = true
	}
	if example != "" {
		schema["examples"] = []any{exampleValue(schema, example)}
	}
}

func exampleValue(schema Schema, example string) any {
	if alternatives, ok := schema["anyOf"].([]Schema); ok {
		// Skel's nullable wrapper always places the declared type first.
		return exampleValue(alternatives[0], example)
	}
	if schema["type"] == "string" {
		var decoded string
		if err := json.Unmarshal([]byte(example), &decoded); err == nil && example != "null" {
			return decoded
		}
		// Numeric-looking examples for strings and decimals remain strings.
		return example
	}
	if raw := jsontext.Value(example); raw.IsValid() {
		return raw
	}
	return example
}
