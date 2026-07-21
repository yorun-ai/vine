package skel

import (
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFacadeBinaryJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Binary `json:"value"`
	}

	data, err := json.Marshal(payload{Value: Binary([]byte("vine"))})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if string(decoded.Value) != "vine" {
		t.Fatalf("unexpected binary value: %q", string(decoded.Value))
	}
}

func TestFacadeUUIDJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value UUID `json:"value"`
	}

	data, err := json.Marshal(payload{Value: NewUUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := decoded.Value.String(); got != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected uuid value: %s", got)
	}
}

func TestFacadeJSONJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value JSON `json:"value"`
	}

	data, err := json.Marshal(payload{Value: JSON(`{"name":"vine"}`)})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := string(decoded.Value); got != `{"name":"vine"}` {
		t.Fatalf("unexpected json value: %s", got)
	}
}

func TestFacadeDecimalJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Decimal `json:"value"`
	}

	data, err := json.Marshal(payload{Value: NewDecimal(decimal.RequireFromString("1.00"))})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"1.00"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !decoded.Value.Equal(decimal.RequireFromString("1.00")) {
		t.Fatalf("unexpected decimal value: %s", decoded.Value.String())
	}
}

func TestFacadeTimestampJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Timestamp `json:"value"`
	}

	data, err := json.Marshal(payload{
		Value: NewTimestamp(time.Date(2026, 5, 4, 13, 14, 15, 123456789, time.FixedZone("CST+8", 8*3600))),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := decoded.Value.Format(time.RFC3339Nano); got != "2026-05-04T05:14:15.123456789Z" {
		t.Fatalf("unexpected timestamp value: %s", got)
	}
}

func TestFacadeLocalDateJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value LocalDate `json:"value"`
	}

	data, err := json.Marshal(payload{
		Value: NewLocalDate(civil.Date{Year: 2026, Month: time.May, Day: 4}),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := decoded.Value.String(); got != "2026-05-04" {
		t.Fatalf("unexpected date value: %s", got)
	}
}

func TestFacadeDurationJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Duration `json:"value"`
	}

	data, err := json.Marshal(payload{
		Value: NewDuration(time.Minute + 500*time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.Duration != time.Minute+500*time.Millisecond {
		t.Fatalf("unexpected duration value: %s", decoded.Value.String())
	}
}
