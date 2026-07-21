package meta

import "testing"

func TestNewInitiatorAcceptsValidValues(t *testing.T) {
	initiator, err := NewInitiator("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "http", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected initiator error: %v", err)
	}
	if initiator.Name() != "demo.service" || initiator.Version() != "1.2.3" || initiator.InstanceId() != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected initiator app content")
	}
	if initiator.Dialer() != "http" {
		t.Fatalf("unexpected dialer: %s", initiator.Dialer())
	}
	if initiator.IpAddr() != "127.0.0.1" {
		t.Fatalf("unexpected ip addr: %v", initiator.IpAddr())
	}
}

func TestNewInitiatorRejectsInvalidApp(t *testing.T) {
	if _, err := NewInitiator("BadName", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "http", "127.0.0.1"); err == nil {
		t.Fatalf("expected invalid app to be rejected")
	}
}

func TestNewInitiatorRejectsInvalidIP(t *testing.T) {
	if _, err := NewInitiator("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "http", "bad-ip"); err == nil {
		t.Fatalf("expected invalid ip to be rejected")
	}
}

func TestNewInitiatorAllowsEmptyIP(t *testing.T) {
	initiator, err := NewInitiator("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "http", "")
	if err != nil {
		t.Fatalf("unexpected initiator error: %v", err)
	}
	if initiator.IpAddr() != "" {
		t.Fatalf("expected empty ip")
	}
}

func TestInitiatorBase64RoundTrip(t *testing.T) {
	initiator, err := NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
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
