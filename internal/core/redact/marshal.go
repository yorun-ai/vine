package redact

import (
	"bytes"
	"encoding"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"
)

var unmarshalRawNumbers = json.WithUnmarshalers(
	json.UnmarshalFromFunc(func(decoder *jsontext.Decoder, value *any) error {
		if decoder.PeekKind() == jsontext.KindNumber {
			*value = jsontext.Value(nil)
		}
		return errors.ErrUnsupported
	}),
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
		if !jsontext.Value(encoded).IsValid() {
			return nil, 0, true, newFailure(
				"decode_json",
				fmt.Errorf("custom JSON marshaler returned invalid JSON"),
			)
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded, unmarshalRawNumbers); err != nil {
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
