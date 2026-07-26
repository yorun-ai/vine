package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

type taggedValue struct {
	Name   string `json:"name"`
	Secret string `json:"customSecret" skel:"sensitive"`
}

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

func TestRenderMasksTaggedFieldsAndSensitiveKeys(t *testing.T) {
	result, err := Render(struct {
		Profile taggedValue       `json:"profile"`
		Meta    map[string]string `json:"meta"`
	}{
		Profile: taggedValue{Name: "Alice", Secret: "tagged-secret"},
		Meta:    map[string]string{"access_token": "key-secret", "visible": "value"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted {
		t.Fatal("expected redacted result")
	}
	if strings.Contains(result.JSON, "tagged-secret") || strings.Contains(result.JSON, "key-secret") {
		t.Fatalf("sensitive value leaked: %s", result.JSON)
	}
	if !strings.Contains(result.JSON, `"customSecret":"<redacted>"`) ||
		!strings.Contains(result.JSON, `"access_token":"<redacted>"`) {
		t.Fatalf("unexpected rendering: %s", result.JSON)
	}
}

func TestRenderMasksExplicitlySensitiveRoot(t *testing.T) {
	result, err := Render(map[string]string{"value": "secret"}, Option{Sensitive: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted || result.JSON != `"<redacted>"` {
		t.Fatalf("unexpected rendering: %#v", result)
	}

	revealed, err := Render(map[string]string{"value": "visible"}, Option{
		Sensitive:       true,
		RevealSensitive: true,
	})
	if err != nil {
		t.Fatalf("render revealed: %v", err)
	}
	if revealed.Redacted || revealed.JSON != `{"value":"visible"}` {
		t.Fatalf("unexpected revealed rendering: %#v", revealed)
	}
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
	digest := sha256.Sum256(binary)
	summary := "<binary:13 bytes sha256=" + hex.EncodeToString(digest[:]) + ">"
	if !strings.Contains(result.JSON, `"token":"visible"`) || !strings.Contains(result.JSON, summary) {
		t.Fatalf("unexpected rendering: %s", result.JSON)
	}
	if strings.Contains(result.JSON, "binary-secret") {
		t.Fatalf("binary value leaked: %s", result.JSON)
	}
	if !result.Redacted {
		t.Fatal("binary summarization should mark the result as redacted")
	}
}

func TestRenderHandlesCycles(t *testing.T) {
	value := map[string]any{}
	value["self"] = value
	result, err := Render(value)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, "<cycle>") {
		t.Fatalf("unexpected cycle rendering: %s", result.JSON)
	}
}

func TestRenderReturnsMarshalerErrors(t *testing.T) {
	if _, err := Render(failingMarshaler{}); err == nil {
		t.Fatal("expected marshaler error")
	}
}

func TestRenderPreservesNumbersAndRedactsKeysFromCustomJSON(t *testing.T) {
	result, err := Render(objectMarshaler{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, `"count":1`) ||
		!strings.Contains(result.JSON, `"token":"<redacted>"`) {
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
