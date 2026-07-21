package meta

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewAppAcceptsValidValues(t *testing.T) {
	app, err := NewApp("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Name() != "demo.service" || app.Version() != "1.2.3" || app.InstanceId() != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected app content: %#v", app)
	}
}

func TestNewAppAcceptsDerivedNameWithSuffix(t *testing.T) {
	app, err := NewApp("demo.worker@demo.runtime", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Name() != "demo.worker@demo.runtime" {
		t.Fatalf("unexpected app name: %s", app.Name())
	}
}

func TestNewAppRejectsInvalidName(t *testing.T) {
	if _, err := NewApp("DemoService", "1.2.3", "123e4567-e89b-12d3-a456-426614174000"); err == nil {
		t.Fatalf("expected invalid name error")
	}
}

func TestNewAppRejectsInvalidVersion(t *testing.T) {
	if _, err := NewApp("demo.service", "latest", "123e4567-e89b-12d3-a456-426614174000"); err == nil {
		t.Fatalf("expected invalid version error")
	}
}

func TestNewAppRejectsInvalidInstanceID(t *testing.T) {
	if _, err := NewApp("demo.service", "1.2.3", "not-a-uuid"); err == nil {
		t.Fatalf("expected invalid instance id error")
	}
}

func TestNewAppRejectsEmptyInstanceID(t *testing.T) {
	if _, err := NewApp("demo.service", "1.2.3", ""); err == nil {
		t.Fatalf("expected empty instance id to be rejected")
	}
}

func TestMustNewAppPanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = MustNewApp("DemoService", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
}

func TestMustNewAppWithRandomIdGeneratesValidInstanceID(t *testing.T) {
	app := MustNewAppWithRandomId("demo.service", "1.2.3")

	if app.Name() != "demo.service" || app.Version() != "1.2.3" {
		t.Fatalf("unexpected app content: %#v", app)
	}
	if err := uuid.Validate(app.InstanceId()); err != nil {
		t.Fatalf("unexpected instance id: %v", err)
	}
	parsed := uuid.MustParse(app.InstanceId())
	if parsed.Version() != 7 {
		t.Fatalf("expected v7 uuid, got v%d", parsed.Version())
	}
}

func TestIsSame(t *testing.T) {
	left, err := NewApp("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	right, err := NewApp("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	other, err := NewApp("demo.service", "1.2.4", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !IsSame(left, right) {
		t.Fatalf("expected apps to be equal")
	}
	if IsSame(left, other) {
		t.Fatalf("expected apps to be different")
	}
	if IsSame(left, nil) {
		t.Fatalf("expected app and nil to be different")
	}
}

func TestAppDelimitedRoundTrip(t *testing.T) {
	app, err := NewApp("demo.service", "1.2.3", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	encoded := EncodeAppToDelimited(app)
	if encoded != "name=demo.service,version=1.2.3,instanceId=123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected app string: %s", encoded)
	}
	got, err := DecodeAppFromDelimited(encoded)
	if err != nil {
		t.Fatalf("DecodeAppFromDelimited() error = %v", err)
	}
	if got.Name() != app.Name() || got.Version() != app.Version() || got.InstanceId() != app.InstanceId() {
		t.Fatalf("unexpected app: %#v", got)
	}
}

func TestDecodeAppFromDelimitedRejectsInvalidValue(t *testing.T) {
	if _, err := DecodeAppFromDelimited("name=demo.service,version=latest,instanceId=123e4567-e89b-12d3-a456-426614174000"); err == nil {
		t.Fatalf("expected invalid version to be rejected")
	}
	if _, err := DecodeAppFromDelimited("name=demo.service,version=1.2.3"); err == nil {
		t.Fatalf("expected missing instance id to be rejected")
	}
}
