package redact

import (
	"errors"
	"strings"
	"testing"
)

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

type objectMarshaler struct{}

func (objectMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"count":1,"token":"secret"}`), nil
}

type taggedMarshaler struct {
	Secret string `json:"customSecret" skel:"sensitive"`
}

func (taggedMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"customSecret":"leaked-by-marshaler"}`), nil
}

type oversizedMarshaler struct{}

func (oversizedMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strings.Repeat("x", 100) + `"`), nil
}

type invalidJSONMarshaler struct{}

func (invalidJSONMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"value":1}{"value":2}`), nil
}

type pointerJSONMarshaler struct{}

func (*pointerJSONMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`"pointer-marshaler"`), nil
}

type nestedSensitiveValue struct {
	Token string `json:"token" skel:"sensitive"`
}

type nestedSensitiveMarshaler struct {
	Value nestedSensitiveValue `json:"value"`
}

func (nestedSensitiveMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"value":{"token":"leaked-by-marshaler"}}`), nil
}

type nestedSensitiveMarker struct {
	Value string `json:"value"`
}

func (nestedSensitiveMarker) SkelSensitive() {}

type nestedMarkerMarshaler struct {
	Value nestedSensitiveMarker `json:"value"`
}

func (nestedMarkerMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"value":{"value":"leaked-by-marshaler"}}`), nil
}

type dynamicSensitiveMarshaler struct {
	Value any `json:"value"`
}

func (dynamicSensitiveMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"value":{"value":"leaked-by-marshaler"}}`), nil
}

func TestRenderRevealSensitiveDoesNotRevealBinary(t *testing.T) {
	binary := []byte("binary-secret")
	result, err := Render(map[string]any{
		"token": "visible",
		"data":  binary,
	}, Option{RevealSensitive: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, `"token":"visible"`) ||
		!strings.Contains(result.JSON, "<binary:13 bytes>") {
		t.Fatalf("unexpected rendering: %s", result.JSON)
	}
	if strings.Contains(result.JSON, "binary-secret") {
		t.Fatalf("binary value leaked: %s", result.JSON)
	}
	if !result.Redacted {
		t.Fatal("binary summarization should mark the result as redacted")
	}
}

func TestRenderReturnsMarshalerErrors(t *testing.T) {
	if _, err := Render(failingMarshaler{}); err == nil {
		t.Fatal("expected marshaler error")
	} else if failureKind(err) != "marshal_json" {
		t.Fatalf("unexpected failure kind: %s", failureKind(err))
	}
}

func TestRenderRejectsTrailingCustomJSON(t *testing.T) {
	if _, err := Render(invalidJSONMarshaler{}); err == nil {
		t.Fatal("expected invalid custom JSON error")
	} else if failureKind(err) != "decode_json" {
		t.Fatalf("unexpected failure kind: %s", failureKind(err))
	}
}

func TestRenderUsesPointerReceiverMarshaler(t *testing.T) {
	result, err := Render(new(pointerJSONMarshaler))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if result.JSON != `"pointer-marshaler"` {
		t.Fatalf("unexpected rendering: %#v", result)
	}
}

func TestRenderTruncatesCustomJSONBeforeProjection(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxOutputBytes = 32
	result, err := Render(oversizedMarshaler{}, Option{Limits: &limits})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Truncated || result.JSON != `"<truncated:json bytes=102>"` {
		t.Fatalf("unexpected custom JSON truncation: %#v", result)
	}
}

func TestRenderPreservesCustomJSONWithoutInferringSensitiveKeys(t *testing.T) {
	result, err := Render(objectMarshaler{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, `"count":1`) ||
		!strings.Contains(result.JSON, `"token":"secret"`) ||
		result.Redacted {
		t.Fatalf("unexpected custom JSON rendering: %s", result.JSON)
	}
}

func TestRenderTagTakesPriorityOverCustomJSON(t *testing.T) {
	result, err := Render(taggedMarshaler{Secret: "tagged-secret"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(result.JSON, "tagged-secret") || strings.Contains(result.JSON, "leaked-by-marshaler") ||
		!strings.Contains(result.JSON, `"customSecret":"<redacted>"`) {
		t.Fatalf("sensitive tag did not take priority: %s", result.JSON)
	}
}

func TestRenderNestedTagTakesPriorityOverOuterCustomJSON(t *testing.T) {
	result, err := Render(nestedSensitiveMarshaler{
		Value: nestedSensitiveValue{Token: "tagged-secret"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(result.JSON, "tagged-secret") ||
		strings.Contains(result.JSON, "leaked-by-marshaler") ||
		!strings.Contains(result.JSON, `"token":"<redacted>"`) {
		t.Fatalf("nested sensitive tag did not take priority: %s", result.JSON)
	}
}

func TestRenderNestedMarkerTakesPriorityOverOuterCustomJSON(t *testing.T) {
	result, err := Render(nestedMarkerMarshaler{
		Value: nestedSensitiveMarker{Value: "marked-secret"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(result.JSON, "marked-secret") ||
		strings.Contains(result.JSON, "leaked-by-marshaler") ||
		!strings.Contains(result.JSON, `"value":"<redacted>"`) {
		t.Fatalf("nested sensitive marker did not take priority: %s", result.JSON)
	}
}

func TestRenderDynamicMarkerTakesPriorityOverOuterCustomJSON(t *testing.T) {
	result, err := Render(dynamicSensitiveMarshaler{
		Value: nestedSensitiveMarker{Value: "marked-secret"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(result.JSON, "marked-secret") ||
		strings.Contains(result.JSON, "leaked-by-marshaler") ||
		!strings.Contains(result.JSON, `"value":"<redacted>"`) {
		t.Fatalf("dynamic sensitive marker did not take priority: %s", result.JSON)
	}
}
