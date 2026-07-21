package meta

import (
	"encoding/base64"
	"strings"
	"testing"

	"go.yorun.ai/vine/util/vcode"
)

type _Base64Payload struct {
	Name string `json:"name"`
}

func TestEncodePayloadToBase64UsesRawURLEncoding(t *testing.T) {
	encoded := encodePayloadToBase64(&_Base64Payload{Name: "demo"})
	if strings.ContainsAny(encoded, "+/=") {
		t.Fatalf("expected raw url base64, got %q", encoded)
	}

	got, err := decodePayloadFromBase64[*_Base64Payload](encoded)
	if err != nil {
		t.Fatalf("decodePayloadFromBase64() error = %v", err)
	}
	if got.Name != "demo" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestDecodePayloadFromBase64RejectsStandardEncoding(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(vcode.MustMarshalJson(&_Base64Payload{Name: "sample"}))
	if !strings.Contains(encoded, "=") {
		t.Fatalf("test fixture expected padded standard base64, got %q", encoded)
	}

	_, err := decodePayloadFromBase64[*_Base64Payload](encoded)
	if err == nil {
		t.Fatalf("expected standard base64 to be rejected")
	}
}
