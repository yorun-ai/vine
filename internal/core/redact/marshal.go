package redact

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"unicode/utf8"
)

func marshalValue(value reflect.Value, maxJSONBytes int) (any, int, bool, error) {
	if !value.IsValid() || !value.CanInterface() {
		return nil, 0, false, nil
	}
	if marshaler, ok := jsonMarshaler(value); ok {
		encoded, err := marshaler.MarshalJSON()
		if err != nil {
			return nil, 0, true, newFailure("marshal_json", err)
		}
		if len(encoded) > maxJSONBytes {
			return nil, len(encoded), true, nil
		}
		if !json.Valid(encoded) {
			return nil, 0, true, newFailure(
				"decode_json",
				fmt.Errorf("custom JSON marshaler returned invalid JSON"),
			)
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			return nil, 0, true, newFailure("decode_json", err)
		}
		return decoded, 0, true, nil
	}
	if marshaler, ok := textMarshaler(value); ok {
		encoded, err := marshaler.MarshalText()
		if err != nil {
			return nil, 0, true, newFailure("marshal_text", err)
		}
		if !utf8.Valid(encoded) {
			encoded = bytes.ToValidUTF8(encoded, []byte("�"))
		}
		return string(encoded), 0, true, nil
	}
	return nil, 0, false, nil
}

func jsonMarshaler(value reflect.Value) (json.Marshaler, bool) {
	if marshaler, ok := value.Interface().(json.Marshaler); ok {
		return marshaler, true
	}
	if value.CanAddr() {
		marshaler, ok := value.Addr().Interface().(json.Marshaler)
		return marshaler, ok
	}
	return nil, false
}

func textMarshaler(value reflect.Value) (encoding.TextMarshaler, bool) {
	if marshaler, ok := value.Interface().(encoding.TextMarshaler); ok {
		return marshaler, true
	}
	if value.CanAddr() {
		marshaler, ok := value.Addr().Interface().(encoding.TextMarshaler)
		return marshaler, ok
	}
	return nil, false
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
	return fmt.Sprintf("<binary:%d bytes>", value.Len())
}
