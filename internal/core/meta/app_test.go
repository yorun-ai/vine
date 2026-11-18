package meta

import (
	"testing"

	"uuid"
)

func TestNewAppRejectsInvalidName(t *testing.T) {
	for _, name := range []string{"DemoService", "demo.worker@demo.runtime", "2worker", "user-service", ".worker", "demo..worker"} {
		if _, err := NewApp(name, "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")); err == nil {
			t.Fatalf("expected invalid name error for %q", name)
		}
	}
}

// A Plot domain name becomes the name of the application that serves it, so
// application names accept the digits a domain name allows.
func TestNewAppAcceptsDigitsInNameSegments(t *testing.T) {
	for _, name := range []string{"worker2", "shop.orders", "team1.app2", "order2.items"} {
		app, err := NewApp(name, "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
		if app.Name() != name {
			t.Fatalf("unexpected app name: %q", app.Name())
		}
	}
}

func TestNewAppRejectsInvalidVersion(t *testing.T) {
	if _, err := NewApp("demo.service", "latest", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")); err == nil {
		t.Fatalf("expected invalid version error")
	}
}

func TestIsValidVersionAcceptsGoModuleFormAndPlainSemver(t *testing.T) {
	for _, version := range []string{
		"0.0.0",
		"1.2.3",
		"v1.2.3",
		"v0.0.0-dev",
		"1.2.3-rc.1",
		"1.2.3+build.5",
	} {
		if !IsValidVersion(version) {
			t.Fatalf("expected valid version for %q", version)
		}
	}
}

func TestIsValidVersionRejectsIncompleteVersions(t *testing.T) {
	for _, version := range []string{
		"",
		"v",
		"1",
		"1.2",
		"v1.2",
		"01.2.3",
		"v01.2.3",
		"1.2.3.4",
		"1.2.3-01",
		"V1.2.3",
		"latest",
	} {
		if IsValidVersion(version) {
			t.Fatalf("expected invalid version for %q", version)
		}
	}
}

func TestDecodeAppRejectsInvalidInstanceID(t *testing.T) {
	if _, err := DecodeAppFromDelimited("name=demo.service,version=1.2.3,instanceId=not-a-uuid"); err == nil {
		t.Fatalf("expected invalid instance id error")
	}
}

func TestDecodeAppRejectsEmptyInstanceID(t *testing.T) {
	if _, err := DecodeAppFromDelimited("name=demo.service,version=1.2.3,instanceId="); err == nil {
		t.Fatalf("expected empty instance id to be rejected")
	}
}

func TestMustNewAppPanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = MustNewApp("DemoService", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
}

func TestMustNewAppWithRandomIdGeneratesValidInstanceID(t *testing.T) {
	app := MustNewAppWithRandomId("demo.service", "1.2.3")

	if app.Name() != "demo.service" || app.Version() != "1.2.3" {
		t.Fatalf("unexpected app content: %#v", app)
	}
	parsed := app.InstanceId()
	if version := parsed[6] >> 4; version != 7 {
		t.Fatalf("expected v7 uuid, got v%d", version)
	}
}

func TestIsSame(t *testing.T) {
	left, err := NewApp("demo.service", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	right, err := NewApp("demo.service", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	other, err := NewApp("demo.service", "1.2.4", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
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
	app, err := NewApp("demo.service", "1.2.3", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"))
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

func TestDecodeAppPreservesLegacyUUIDForms(t *testing.T) {
	canonical := "123e4567-e89b-12d3-a456-426614174000"
	want := uuid.MustParse(canonical)
	for _, text := range []string{canonical, "123E4567-E89B-12D3-A456-426614174000", "{123e4567-e89b-12d3-a456-426614174000}", "urn:uuid:" + canonical, "123e4567e89b12d3a456426614174000"} {
		t.Run(text, func(t *testing.T) {
			app, err := DecodeAppFromDelimited("name=demo.service,version=v1.2.3,instanceId=" + text)
			if err != nil {
				t.Fatal(err)
			}
			if app.InstanceId() != want || app.Version() != "v1.2.3" {
				t.Fatalf("legacy identity changed: %v", app)
			}
			if got := EncodeAppToDelimited(app); got != "name=demo.service,version=v1.2.3,instanceId="+canonical {
				t.Fatalf("UUID must remain text on the wire: %s", got)
			}
		})
	}
}

func TestAppAcceptsZeroUUID(t *testing.T) {
	// The previous string constructor accepted the all-zero UUID as well.
	app, err := NewApp("demo.service", "1.2.3", uuid.UUID{})
	if err != nil || app.InstanceId() != (uuid.UUID{}) {
		t.Fatalf("zero UUID compatibility: %v, %v", app, err)
	}
}
