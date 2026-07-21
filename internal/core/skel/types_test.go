package skel

import (
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.yorun.ai/vine/util/vcode"
)

func TestBinaryJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Binary `json:"value"`
	}

	input := payload{Value: Binary([]byte("hello"))}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"aGVsbG8="}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if string(decoded.Value) != "hello" {
		t.Fatalf("unexpected binary value: %q", string(decoded.Value))
	}
}

func TestBinaryUnmarshalJSONNull(t *testing.T) {
	var value Binary
	if err := json.Unmarshal([]byte("null"), &value); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil binary, got %#v", []byte(value))
	}
}

func TestBinaryCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value Binary `cbor:"value"`
	}

	input := payload{Value: Binary([]byte("hello"))}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if string(decoded.Value) != "hello" {
		t.Fatalf("unexpected binary value: %q", string(decoded.Value))
	}
}

func TestUUIDJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value UUID `json:"value"`
	}

	input := payload{Value: NewUUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"550e8400-e29b-41d4-a716-446655440000"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.UUID != input.Value.UUID {
		t.Fatalf("unexpected uuid value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestUUIDCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value UUID `cbor:"value"`
	}

	input := payload{Value: NewUUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if decoded.Value.UUID != input.Value.UUID {
		t.Fatalf("unexpected uuid value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestJSONJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value JSON `json:"value"`
	}

	input := payload{Value: JSON(`{"name":"vine","count":2}`)}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"{\"name\":\"vine\",\"count\":2}"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value != input.Value {
		t.Fatalf("unexpected json value: got=%s want=%s", decoded.Value, input.Value)
	}
}

func TestJSONCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value JSON `cbor:"value"`
	}

	input := payload{Value: JSON(`{"name":"vine","count":2}`)}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if decoded.Value != input.Value {
		t.Fatalf("unexpected json value: got=%s want=%s", decoded.Value, input.Value)
	}
}

func TestDecimalJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Decimal `json:"value"`
	}

	input := payload{Value: NewDecimal(decimal.RequireFromString("1.00"))}
	data, err := json.Marshal(input)
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
	if !decoded.Value.Equal(input.Value.Decimal) {
		t.Fatalf("unexpected decimal value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestDecimalCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value Decimal `cbor:"value"`
	}

	input := payload{Value: NewDecimal(decimal.RequireFromString("1.00"))}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}
	var encoded map[string]string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("Unmarshal() encoded cbor error = %v", err)
	}
	if got := encoded["value"]; got != "1.00" {
		t.Fatalf("unexpected cbor decimal text: %s", got)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if !decoded.Value.Equal(input.Value.Decimal) {
		t.Fatalf("unexpected decimal value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestTimestampJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Timestamp `json:"value"`
	}

	raw := time.Date(2026, 5, 4, 13, 14, 15, 123456789, time.FixedZone("CST+8", 8*3600))
	input := payload{Value: NewTimestamp(raw)}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"2026-05-04T05:14:15.123456789Z"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !decoded.Value.Equal(raw.UTC()) {
		t.Fatalf("unexpected timestamp value: got=%s want=%s", decoded.Value.Format(timestampLayout), raw.UTC().Format(timestampLayout))
	}
}

func TestTimestampCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value Timestamp `cbor:"value"`
	}

	raw := time.Date(2026, 5, 4, 13, 14, 15, 123456789, time.FixedZone("CST+8", 8*3600))
	input := payload{Value: NewTimestamp(raw)}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}
	got, err := cbor.Diagnose(data)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got != "{"+`"value": "2026-05-04T05:14:15.123456789Z"`+"}" {
		t.Fatalf("unexpected cbor diagnose: %s", got)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if !decoded.Value.Equal(raw.UTC()) {
		t.Fatalf("unexpected timestamp value: got=%s want=%s", decoded.Value.Format(timestampLayout), raw.UTC().Format(timestampLayout))
	}
}

func TestNewTimestampNowReturnsUTCValue(t *testing.T) {
	value := NewTimestampNow()
	if value.Location() != time.UTC {
		t.Fatalf("unexpected timestamp location: got=%s want=%s", value.Location(), time.UTC)
	}
	if value.IsZero() {
		t.Fatal("unexpected zero timestamp")
	}
}

func TestLocalDateJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value LocalDate `json:"value"`
	}

	input := payload{Value: NewLocalDate(civil.Date{Year: 2026, Month: time.May, Day: 4})}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"2026-05-04"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.Date != input.Value.Date {
		t.Fatalf("unexpected date value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestLocalDateCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value LocalDate `cbor:"value"`
	}

	input := payload{Value: NewLocalDate(civil.Date{Year: 2026, Month: time.May, Day: 4})}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}
	got, err := cbor.Diagnose(data)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got != "{"+`"value": "2026-05-04"`+"}" {
		t.Fatalf("unexpected cbor diagnose: %s", got)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if decoded.Value.Date != input.Value.Date {
		t.Fatalf("unexpected date value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestLocalTimeJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value LocalTime `json:"value"`
	}

	input := payload{Value: NewLocalTime(civil.Time{Hour: 9, Minute: 30, Second: 0, Nanosecond: 123000000})}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"09:30:00.123000000"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.Time != input.Value.Time {
		t.Fatalf("unexpected localtime value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestLocalDateTimeJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value LocalDateTime `json:"value"`
	}

	input := payload{Value: NewLocalDateTime(civil.DateTime{
		Date: civil.Date{Year: 2026, Month: time.May, Day: 4},
		Time: civil.Time{Hour: 9, Minute: 30, Second: 0, Nanosecond: 123000000},
	})}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"2026-05-04T09:30:00.123000000"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.DateTime != input.Value.DateTime {
		t.Fatalf("unexpected localdatetime value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestDurationJSONRoundTrip(t *testing.T) {
	type payload struct {
		Value Duration `json:"value"`
	}

	input := payload{Value: NewDuration(time.Hour + 2*time.Minute + 3*time.Second)}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(data); got != `{"value":"1h2m3s"}` {
		t.Fatalf("unexpected json: %s", got)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Value.Duration != input.Value.Duration {
		t.Fatalf("unexpected duration value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}

func TestDurationCBORRoundTrip(t *testing.T) {
	type payload struct {
		Value Duration `cbor:"value"`
	}

	input := payload{Value: NewDuration(250 * time.Millisecond)}
	data, err := vcode.MarshalCbor(input)
	if err != nil {
		t.Fatalf("MarshalCbor() error = %v", err)
	}
	got, err := cbor.Diagnose(data)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got != "{"+`"value": "250ms"`+"}" {
		t.Fatalf("unexpected cbor diagnose: %s", got)
	}

	decoded, err := vcode.UnmarshalCbor[payload](data)
	if err != nil {
		t.Fatalf("UnmarshalCbor() error = %v", err)
	}
	if decoded.Value.Duration != input.Value.Duration {
		t.Fatalf("unexpected duration value: got=%s want=%s", decoded.Value.String(), input.Value.String())
	}
}
