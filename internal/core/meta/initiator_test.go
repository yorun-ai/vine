package meta

import (
	"encoding/base64"
	"testing"
	"uuid"
)

func TestInitiatorLegacyWireCompatibility(t *testing.T) {
	const payload = `{"name":"demo.service","version":"v1.2.3","instanceId":"123e4567-e89b-12d3-a456-426614174000","dialer":"https","ipAddr":"127.0.0.1"}`
	wire := base64.RawURLEncoding.EncodeToString([]byte(payload))
	initiator, err := DecodeInitiatorFromBase64(wire)
	if err != nil {
		t.Fatal(err)
	}
	if initiator.InstanceId() != uuid.MustParse("123e4567-e89b-12d3-a456-426614174000") {
		t.Fatal("legacy initiator UUID changed")
	}
	if got := EncodeInitiatorToBase64(initiator); got != wire {
		t.Fatalf("initiator wire encoding changed: %s", got)
	}
	for _, text := range []string{"", "not-a-uuid"} {
		bad := base64.RawURLEncoding.EncodeToString([]byte(`{"name":"demo.service","version":"1.2.3","instanceId":"` + text + `"}`))
		if _, err := DecodeInitiatorFromBase64(bad); err == nil {
			t.Fatalf("accepted invalid UUID text %q", text)
		}
	}
}

func TestNewInitiatorRejectsInvalidApp(t *testing.T) {
	if _, err := NewInitiator("BadName", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "http", "127.0.0.1"); err == nil {
		t.Fatalf("expected invalid app to be rejected")
	}
}

func TestNewInitiatorRejectsInvalidIP(t *testing.T) {
	if _, err := NewInitiator("demo.service", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "http", "bad-ip"); err == nil {
		t.Fatalf("expected invalid ip to be rejected")
	}
}

func TestNewInitiatorAllowsEmptyIP(t *testing.T) {
	initiator, err := NewInitiator("demo.service", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "http", "")
	if err != nil {
		t.Fatalf("unexpected initiator error: %v", err)
	}
	if initiator.IpAddr() != "" {
		t.Fatalf("expected empty ip")
	}
}

func TestInitiatorBase64RoundTrip(t *testing.T) {
	initiator, err := NewInitiator("portal.app", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}

	got, err := DecodeInitiatorFromBase64(EncodeInitiatorToBase64(initiator))
	if err != nil {
		t.Fatalf("DecodeInitiatorFromBase64() error = %v", err)
	}
	if got == nil || got.Name() != initiator.Name() || got.Dialer() != initiator.Dialer() {
		t.Fatalf("unexpected initiator: %#v", got)
	}
}

func TestDecodeInitiatorFromEmptyBase64ReturnsNil(t *testing.T) {
	got, err := DecodeInitiatorFromBase64("")
	if err != nil {
		t.Fatalf("DecodeInitiatorFromBase64() error = %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil initiator, got %#v", got)
	}
}
