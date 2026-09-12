package skel

import (
	"encoding/json/v2"
	"testing"
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	"github.com/shopspring/decimal"
)

type facadeSensitiveValue struct{}

func (facadeSensitiveValue) SkelSensitive() {}

var _ Sensitive = facadeSensitiveValue{}

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
		"":      `{"resourceSkelName":"app.User","actionName":"read"}`,
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
		if check.CodeArgumentName == "" {
			if _, exists := fields["codeArgumentName"]; exists {
				t.Fatalf("legacy schema should omit codeArgumentName: %s", encoded)
			}
		} else if fields["codeArgumentName"] != "code_" {
			t.Fatalf("custom argument name lost in schema JSON: %s", encoded)
		}
	}
}
