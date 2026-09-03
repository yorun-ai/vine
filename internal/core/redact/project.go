package redact

import (
	"encoding/json/jsontext"
	"fmt"
	"maps"
	"math"
	"reflect"
	"sort"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
)

const redactedValue = "<redacted>"

type _Visit struct {
	kind reflect.Kind
	ptr  uintptr
}

type _ProjectionState struct {
	option    Option
	limits    Limits
	visiting  map[_Visit]struct{}
	redacted  bool
	truncated bool
	nodes     int
}

func newProjectionState(option Option, limits Limits) *_ProjectionState {
	return &_ProjectionState{
		option:   option,
		limits:   limits,
		visiting: map[_Visit]struct{}{},
	}
}

func (s *_ProjectionState) project(value reflect.Value, depth int) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}
		if value.Kind() == reflect.Pointer {
			visit := _Visit{kind: value.Kind(), ptr: value.Pointer()}
			if _, exists := s.visiting[visit]; exists {
				return "<cycle>", nil
			}
			s.visiting[visit] = struct{}{}
			defer delete(s.visiting, visit)
		}
		value = value.Elem()
	}
	if !s.option.RevealSensitive && value.CanInterface() {
		if _, sensitive := value.Interface().(skel.Sensitive); sensitive {
			s.redacted = true
			return redactedValue, nil
		}
	}
	if depth > s.limits.MaxDepth {
		return s.truncatedValue("depth", "limit", s.limits.MaxDepth), nil
	}
	s.nodes++
	if s.nodes > s.limits.MaxNodes {
		return s.truncatedValue("nodes", "limit", s.limits.MaxNodes), nil
	}

	if !s.option.RevealSensitive && hasSensitiveMetadata(value.Type()) {
		switch value.Kind() {
		case reflect.Struct:
			return s.projectStruct(value, depth)
		case reflect.Map:
			return s.projectMap(value, depth)
		case reflect.Array, reflect.Slice:
			return s.projectList(value, depth)
		}
	}
	if raw, ok := value.Interface().(jsontext.Value); ok && raw.Kind() == jsontext.KindNumber {
		return raw, nil
	}
	if isBinary(value) {
		s.redacted = true
		return binarySummary(value), nil
	}
	if marshaled, oversizedBytes, ok, err := marshalValue(value, s.limits.MaxOutputBytes); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if oversizedBytes > 0 {
			return s.truncatedValue("json", "bytes", oversizedBytes), nil
		}
		return s.project(reflect.ValueOf(marshaled), depth+1)
	}
	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.String:
		text := strings.ToValidUTF8(value.String(), "�")
		if len(text) > s.limits.MaxStringBytes {
			return s.truncatedValue("string", "bytes", len(text)), nil
		}
		return text, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), nil
	case reflect.Float32, reflect.Float64:
		floatValue := value.Float()
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return "<non-finite-number>", nil
		}
		return floatValue, nil
	case reflect.Complex64, reflect.Complex128:
		return fmt.Sprintf("<complex:%v>", value.Complex()), nil
	case reflect.Struct:
		return s.projectStruct(value, depth)
	case reflect.Map:
		return s.projectMap(value, depth)
	case reflect.Slice, reflect.Array:
		return s.projectList(value, depth)
	case reflect.Invalid:
		return nil, nil
	default:
		return "<" + value.Type().String() + ">", nil
	}
}

func (s *_ProjectionState) truncatedValue(kind string, sizeName string, size int) string {
	s.truncated = true
	return fmt.Sprintf("<truncated:%s %s=%d>", kind, sizeName, size)
}

func hasSensitiveMetadata(valueType reflect.Type) bool {
	return typeHasSensitiveMetadata(valueType, make(map[reflect.Type]struct{}))
}

func typeHasSensitiveMetadata(valueType reflect.Type, visiting map[reflect.Type]struct{}) bool {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	if _, exists := visiting[valueType]; exists {
		return false
	}
	visiting[valueType] = struct{}{}
	defer delete(visiting, valueType)

	sensitiveType := reflect.TypeFor[skel.Sensitive]()
	if valueType.Implements(sensitiveType) ||
		(valueType.Kind() != reflect.Pointer && reflect.PointerTo(valueType).Implements(sensitiveType)) {
		return true
	}

	switch valueType.Kind() {
	case reflect.Interface:
		// Dynamic values may implement skel.Sensitive at runtime. Composite
		// containers with interface members must be projected before honoring
		// an outer custom marshaler.
		return true
	case reflect.Struct:
		for field := range valueType.Fields() {
			if field.PkgPath != "" || field.Tag.Get("json") == "-" {
				continue
			}
			if field.Tag.Get("skel") == "sensitive" ||
				typeHasSensitiveMetadata(field.Type, visiting) {
				return true
			}
		}
	case reflect.Array, reflect.Slice:
		return typeHasSensitiveMetadata(valueType.Elem(), visiting)
	case reflect.Map:
		return typeHasSensitiveMetadata(valueType.Key(), visiting) ||
			typeHasSensitiveMetadata(valueType.Elem(), visiting)
	}
	return false
}

func (s *_ProjectionState) projectStruct(value reflect.Value, depth int) (any, error) {
	result := map[string]any{}
	for index := range value.NumField() {
		fieldInfo := value.Type().Field(index)
		if fieldInfo.PkgPath != "" {
			continue
		}
		name, embedded, skip := fieldName(fieldInfo)
		if skip {
			continue
		}
		if !s.option.RevealSensitive && fieldInfo.Tag.Get("skel") == "sensitive" {
			s.redacted = true
			result[name] = redactedValue
			continue
		}
		fieldValue := value.Field(index)
		if embedded {
			projected, err := s.project(fieldValue, depth+1)
			if err != nil {
				return nil, err
			}
			if object, ok := projected.(map[string]any); ok {
				maps.Copy(result, object)
			}
			continue
		}
		projected, err := s.project(fieldValue, depth+1)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_ProjectionState) projectMap(value reflect.Value, depth int) (any, error) {
	if value.IsNil() {
		return nil, nil
	}
	if value.Len() > s.limits.MaxCollectionItems {
		return s.truncatedValue("map", "entries", value.Len()), nil
	}
	visit := _Visit{kind: value.Kind(), ptr: value.Pointer()}
	if _, exists := s.visiting[visit]; exists {
		return map[string]any{"_value": "<cycle>"}, nil
	}
	s.visiting[visit] = struct{}{}
	defer delete(s.visiting, visit)

	if value.Type().Key().Kind() != reflect.String {
		return map[string]any{"_value": "<unsupported-map-key>"}, nil
	}
	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	result := make(map[string]any, len(keys))
	for _, mapKey := range keys {
		name := mapKey.String()
		projected, err := s.project(value.MapIndex(mapKey), depth+1)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_ProjectionState) projectList(value reflect.Value, depth int) (any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
	}
	if value.Len() > s.limits.MaxCollectionItems {
		return s.truncatedValue("list", "items", value.Len()), nil
	}
	var visit _Visit
	if value.Kind() == reflect.Slice {
		visit = _Visit{kind: value.Kind(), ptr: value.Pointer()}
		if _, exists := s.visiting[visit]; exists {
			return []any{"<cycle>"}, nil
		}
		s.visiting[visit] = struct{}{}
		defer delete(s.visiting, visit)
	}
	result := make([]any, value.Len())
	for index := range value.Len() {
		projected, err := s.project(value.Index(index), depth+1)
		if err != nil {
			return nil, err
		}
		result[index] = projected
	}
	return result, nil
}

func fieldName(field reflect.StructField) (name string, embedded bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	name, _, _ = strings.Cut(tag, ",")
	if name != "" {
		return name, false, false
	}
	if field.Anonymous {
		return "", true, false
	}
	return field.Name, false, false
}
