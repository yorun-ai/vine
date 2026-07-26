package redact

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

const redactedValue = "<redacted>"

// Option controls how a value is rendered.
type Option struct {
	// RevealSensitive disables sensitive field and key masking. Binary values
	// are still replaced with summaries.
	RevealSensitive bool
	// Sensitive treats the root value as sensitive as a whole.
	Sensitive bool
}

// Result is the JSON-safe rendering of a value.
type Result struct {
	// JSON contains the rendered value without a trailing newline.
	JSON string
	// Redacted reports whether at least one sensitive or binary value was replaced.
	Redacted bool
}

// Render converts value to JSON while masking sensitive fields and keys.
func Render(value any, options ...Option) (result Result, err error) {
	if len(options) > 1 {
		return Result{}, fmt.Errorf("redact.Render accepts at most one Option")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			result = Result{}
			err = fmt.Errorf("render redacted value: %v", recovered)
		}
	}()
	option := Option{}
	if len(options) > 0 {
		option = options[0]
	}
	if option.Sensitive && !option.RevealSensitive {
		return Result{JSON: `"<redacted>"`, Redacted: true}, nil
	}
	state := new(_ProjectionState{
		option:   option,
		visiting: map[_Visit]struct{}{},
	})
	projected, err := state.project(reflect.ValueOf(value), "")
	if err != nil {
		return Result{}, err
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(projected); err != nil {
		return Result{}, err
	}
	return Result{
		JSON:     strings.TrimSuffix(encoded.String(), "\n"),
		Redacted: state.redacted,
	}, nil
}

type _Visit struct {
	kind reflect.Kind
	ptr  uintptr
}

type _ProjectionState struct {
	option   Option
	visiting map[_Visit]struct{}
	redacted bool
}

type _Sensitive interface {
	SkelSensitive()
}

func (s *_ProjectionState) project(value reflect.Value, key string) (any, error) {
	if key != "" && !s.option.RevealSensitive && isSensitiveKey(key) {
		s.redacted = true
		return redactedValue, nil
	}
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
		if _, sensitive := value.Interface().(_Sensitive); sensitive {
			s.redacted = true
			return redactedValue, nil
		}
	}

	if value.Kind() == reflect.Struct && !s.option.RevealSensitive && hasSensitiveField(value.Type()) {
		return s.projectStruct(value)
	}
	if isBinary(value) {
		s.redacted = true
		return binarySummary(value), nil
	}
	if marshaled, ok, err := marshalValue(value); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return s.project(reflect.ValueOf(marshaled), key)
	}
	if value.CanInterface() {
		if number, ok := value.Interface().(json.Number); ok {
			return number, nil
		}
	}

	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.String:
		return strings.ToValidUTF8(value.String(), "�"), nil
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
		return s.projectStruct(value)
	case reflect.Map:
		return s.projectMap(value)
	case reflect.Slice, reflect.Array:
		return s.projectList(value)
	case reflect.Invalid:
		return nil, nil
	default:
		return "<" + value.Type().String() + ">", nil
	}
}

func hasSensitiveField(valueType reflect.Type) bool {
	for index := range valueType.NumField() {
		field := valueType.Field(index)
		if field.PkgPath == "" && field.Tag.Get("skel") == "sensitive" {
			return true
		}
	}
	return false
}

func (s *_ProjectionState) projectStruct(value reflect.Value) (map[string]any, error) {
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
			projected, err := s.project(fieldValue, "")
			if err != nil {
				return nil, err
			}
			if object, ok := projected.(map[string]any); ok {
				for embeddedName, embeddedValue := range object {
					result[embeddedName] = embeddedValue
				}
			}
			continue
		}
		projected, err := s.project(fieldValue, name)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_ProjectionState) projectMap(value reflect.Value) (map[string]any, error) {
	if value.IsNil() {
		return nil, nil
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
		if !s.option.RevealSensitive && isSensitiveKey(name) {
			s.redacted = true
			result[name] = redactedValue
			continue
		}
		projected, err := s.project(value.MapIndex(mapKey), name)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_ProjectionState) projectList(value reflect.Value) ([]any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
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
		projected, err := s.project(value.Index(index), "")
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

func marshalValue(value reflect.Value) (any, bool, error) {
	if !value.IsValid() || !value.CanInterface() {
		return nil, false, nil
	}
	if marshaler, ok := value.Interface().(json.Marshaler); ok {
		encoded, err := marshaler.MarshalJSON()
		if err != nil {
			return nil, true, err
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			return nil, true, err
		}
		return decoded, true, nil
	}
	if marshaler, ok := value.Interface().(encoding.TextMarshaler); ok {
		encoded, err := marshaler.MarshalText()
		if err != nil {
			return nil, true, err
		}
		if !utf8.Valid(encoded) {
			encoded = bytes.ToValidUTF8(encoded, []byte("�"))
		}
		return string(encoded), true, nil
	}
	return nil, false, nil
}

func isBinary(value reflect.Value) bool {
	if value.Kind() == reflect.Slice {
		return value.Type().Elem().Kind() == reflect.Uint8
	}
	if value.Kind() != reflect.Array || value.Type().Elem().Kind() != reflect.Uint8 {
		return false
	}
	_, jsonMarshaler := value.Interface().(json.Marshaler)
	_, textMarshaler := value.Interface().(encoding.TextMarshaler)
	return !jsonMarshaler && !textMarshaler
}

func binarySummary(value reflect.Value) string {
	hasher := sha256.New()
	if value.Kind() == reflect.Slice {
		_, _ = hasher.Write(value.Bytes())
	} else {
		var chunk [4096]byte
		for offset := 0; offset < value.Len(); {
			length := min(len(chunk), value.Len()-offset)
			for index := range length {
				chunk[index] = byte(value.Index(offset + index).Uint())
			}
			_, _ = hasher.Write(chunk[:length])
			offset += length
		}
	}
	return fmt.Sprintf("<binary:%d bytes sha256=%s>", value.Len(), hex.EncodeToString(hasher.Sum(nil)))
}

func isSensitiveKey(key string) bool {
	normalized := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return -1
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, key)
	switch normalized {
	case "password", "passwd", "pwd", "token", "accesstoken", "refreshtoken", "secret",
		"authorization", "cookie", "setcookie", "apikey", "privatekey", "credential":
		return true
	default:
		return false
	}
}
