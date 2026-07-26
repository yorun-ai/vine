package redact_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.yorun.ai/vine/core/redact"
	"go.yorun.ai/vine/core/skel"
)

type sensitiveCredential struct {
	Token string `json:"token"`
}

func (sensitiveCredential) SkelSensitive() {}

func TestRenderPublicAPI(t *testing.T) {
	result, err := redact.Render(struct {
		Password string `json:"password" skel:"sensitive"`
	}{Password: "secret"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted || strings.Contains(result.JSON, "secret") {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRenderSensitiveValueAsAWhole(t *testing.T) {
	var _ skel.Sensitive = sensitiveCredential{}
	result, err := redact.Render(struct {
		Credential sensitiveCredential `json:"credential"`
	}{Credential: sensitiveCredential{Token: "secret"}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted || result.JSON != `{"credential":"<redacted>"}` {
		t.Fatalf("unexpected result: %#v", result)
	}

	revealed, err := redact.Render(sensitiveCredential{Token: "secret"}, redact.Option{RevealSensitive: true})
	if err != nil {
		t.Fatalf("render revealed: %v", err)
	}
	if revealed.Redacted || revealed.JSON != `{"token":"secret"}` {
		t.Fatalf("unexpected revealed result: %#v", revealed)
	}
}

func TestRenderSkelScalars(t *testing.T) {
	result, err := redact.Render(struct {
		ID   skel.UUID   `json:"id"`
		Data skel.Binary `json:"data"`
	}{
		ID:   skel.NewUUID(uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")),
		Data: skel.Binary("secret"),
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, `"id":"123e4567-e89b-12d3-a456-426614174000"`) {
		t.Fatalf("UUID should keep its scalar representation: %s", result.JSON)
	}
	if strings.Contains(result.JSON, "secret") || !strings.Contains(result.JSON, "<binary:6 bytes sha256=") {
		t.Fatalf("binary should be summarized: %s", result.JSON)
	}
}
