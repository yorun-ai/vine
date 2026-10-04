package skel

import (
	"encoding/json/v2"
	"strings"
	"testing"
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	"github.com/shopspring/decimal"
)

func TestExtensionServiceSchemaJSON(t *testing.T) {
	var service ServiceSchema
	if err := json.Unmarshal([]byte(`{"ext":true,"authMode":"noauth"}`), &service); err != nil {
		t.Fatal(err)
	}
	if !service.Ext || service.ClientApi() {
		t.Fatalf("extension lost its direction or was classified as an API: %+v", service)
	}
	encoded, err := json.Marshal(service)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ext":true`) {
		t.Fatalf("extension flag lost in JSON: %s", encoded)
	}
	encoded, err = json.Marshal(ServiceSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"ext"`) {
		t.Fatalf("zero extension flag must be omitted: %s", encoded)
	}
}

type facadeSensitiveValue struct{}

func (facadeSensitiveValue) SkelSensitive() {}

var _ Sensitive = facadeSensitiveValue{}

func TestExtensionEventSchemaJSON(t *testing.T) {
	var event EventSchema
	if err := json.Unmarshal([]byte(`{"ext":true}`), &event); err != nil {
		t.Fatal(err)
	}
	if !event.Ext {
		t.Fatal("extension event flag lost during decoding")
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ext":true`) {
		t.Fatalf("extension flag lost: %s", encoded)
	}
	encoded, err = json.Marshal(EventSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"ext"`) {
		t.Fatalf("zero extension flag must be omitted: %s", encoded)
	}
}

// Public constructors must preserve their arguments; scalar encoding is tested internally.
func TestFacadeScalarConstructors(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	amount := decimal.RequireFromString("1.00")
	instant := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	date := civil.Date{Year: 2026, Month: time.January, Day: 15}
	if NewUUID(id).UUID != id || !NewDecimal(amount).Equal(amount) ||
		!NewTimestamp(instant).Equal(instant) || NewDuration(time.Second).Duration != time.Second ||
		NewLocalDate(date).Date != date {
		t.Fatal("facade scalar constructor changed its input")
	}
}

func TestPermCheckInvocationCodeArgumentNameJSON(t *testing.T) {
	for name, input := range map[string]string{
		"code":  `{"resourceSkelName":"app.User","actionName":"read","codeArgumentName":"code"}`,
		"code_": `{"resourceSkelName":"app.User","actionName":"read","codeArgumentName":"code_"}`,
	} {
		var check PermCheckInvocation
		if err := json.Unmarshal([]byte(input), &check); err != nil {
			t.Fatal(err)
		}
		if check.CodeArgumentName != name {
			t.Fatalf("decoded argument name = %q, want %q", check.CodeArgumentName, name)
		}
		encoded, err := json.Marshal(check)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["codeArgumentName"] != name {
			t.Fatalf("argument name lost in schema JSON: %s", encoded)
		}
	}
}

func TestAuthModeCompatibility(t *testing.T) {
	if string(AuthModeAuth) != "auth" || string(AuthModeNoAuth) != "noauth" {
		t.Fatal("legacy constant values must remain compatible with old generated code")
	}
	if string(AuthModeOptional) != "optional" || string(AuthModeAnonymous) != "anonymous" {
		t.Fatal("unexpected authentication mode values")
	}
}
