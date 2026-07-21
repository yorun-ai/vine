package skel

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"cloud.google.com/go/civil"
	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Shared skel scalar extensions follow a "CBOR is the binary transport form of
// JSON" rule. For most types we keep the CBOR payload shape aligned with JSON,
// even when a more compact CBOR-native representation would be possible,
// because matching shapes are easier to reason about across Go/TS runtimes and
// keep protocol semantics stable. Binary is the intentional exception: JSON has
// no native bytes type, so it uses base64 strings while CBOR uses raw bytes.

const timestampLayout = time.RFC3339Nano

// Decimal is the shared skel decimal type.
// It is encoded as a decimal string in both JSON and CBOR.
type Decimal struct {
	decimal.Decimal
}

func NewDecimal(value decimal.Decimal) Decimal {
	return Decimal{Decimal: value}
}

// encodeString keeps the decimal scale in the wire representation. decimal.String()
// normalizes trailing zeros, so values like "1.00" would otherwise become "1".
func (d Decimal) encodeString() string {
	exponent := d.Exponent()
	if exponent < 0 {
		return d.StringFixed(-exponent)
	}
	return d.String()
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.encodeString())
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Decimal = decimal.Decimal{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := decimal.NewFromString(encoded)
	if err != nil {
		return err
	}
	d.Decimal = decoded
	return nil
}

func (d Decimal) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(d.encodeString())
}

func (d *Decimal) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		d.Decimal = decimal.Decimal{}
		return nil
	}

	decoded, err := decimal.NewFromString(*encoded)
	if err != nil {
		return err
	}
	d.Decimal = decoded
	return nil
}

// Binary

type Binary []byte

type PermissionCode string

func (b Binary) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.StdEncoding.EncodeToString([]byte(b)))
}

func (b *Binary) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*b = nil
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	*b = Binary(decoded)
	return nil
}

func (b Binary) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]byte(b))
}

func (b *Binary) UnmarshalCBOR(data []byte) error {
	var decoded []byte
	if err := cbor.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*b = Binary(decoded)
	return nil
}

// Timestamp is the shared skel timestamp type.
// It is encoded as an RFC3339Nano string in both JSON and CBOR.
type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t.UTC()}
}

func NewTimestampNow() Timestamp {
	return NewTimestamp(time.Now())
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Format(timestampLayout))
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := time.Parse(timestampLayout, encoded)
	if err != nil {
		return err
	}
	t.Time = decoded.UTC()
	return nil
}

func (t Timestamp) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(t.UTC().Format(timestampLayout))
}

func (t *Timestamp) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		t.Time = time.Time{}
		return nil
	}

	decoded, err := time.Parse(timestampLayout, *encoded)
	if err != nil {
		return err
	}
	t.Time = decoded.UTC()
	return nil
}

// Duration is the shared skel duration type.
// It is encoded as a time.ParseDuration-compatible string in both JSON and CBOR.
type Duration struct {
	time.Duration
}

func NewDuration(d time.Duration) Duration {
	return Duration{Duration: d}
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Duration = 0
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := time.ParseDuration(encoded)
	if err != nil {
		return err
	}
	d.Duration = decoded
	return nil
}

func (d Duration) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(d.String())
}

func (d *Duration) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		d.Duration = 0
		return nil
	}

	decoded, err := time.ParseDuration(*encoded)
	if err != nil {
		return err
	}
	d.Duration = decoded
	return nil
}

// LocalDate is the shared skel local date type.
// It is encoded as an RFC3339 full-date string in both JSON and CBOR.
type LocalDate struct {
	civil.Date
}

func NewLocalDate(date civil.Date) LocalDate {
	return LocalDate{Date: date}
}

func NewLocalDateOf(t time.Time) LocalDate {
	return NewLocalDate(civil.DateOf(t))
}

func (d LocalDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *LocalDate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.Date = civil.Date{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := civil.ParseDate(encoded)
	if err != nil {
		return err
	}
	d.Date = decoded
	return nil
}

func (d LocalDate) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(d.String())
}

func (d *LocalDate) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		d.Date = civil.Date{}
		return nil
	}

	decoded, err := civil.ParseDate(*encoded)
	if err != nil {
		return err
	}
	d.Date = decoded
	return nil
}

// LocalTime is the shared skel local time type.
// It is encoded as an RFC3339 partial-time string in both JSON and CBOR.
type LocalTime struct {
	civil.Time
}

func NewLocalTime(clock civil.Time) LocalTime {
	return LocalTime{Time: clock}
}

func NewLocalTimeOf(t time.Time) LocalTime {
	return NewLocalTime(civil.TimeOf(t))
}

func (t LocalTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = civil.Time{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := civil.ParseTime(encoded)
	if err != nil {
		return err
	}
	t.Time = decoded
	return nil
}

func (t LocalTime) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(t.String())
}

func (t *LocalTime) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		t.Time = civil.Time{}
		return nil
	}

	decoded, err := civil.ParseTime(*encoded)
	if err != nil {
		return err
	}
	t.Time = decoded
	return nil
}

// LocalDateTime is the shared skel local datetime type.
// It is encoded as an RFC3339 date-time string without timezone in both JSON and CBOR.
type LocalDateTime struct {
	civil.DateTime
}

func NewLocalDateTime(dateTime civil.DateTime) LocalDateTime {
	return LocalDateTime{DateTime: dateTime}
}

func NewLocalDateTimeOf(t time.Time) LocalDateTime {
	return NewLocalDateTime(civil.DateTimeOf(t))
}

func (dt LocalDateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(dt.String())
}

func (dt *LocalDateTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		dt.DateTime = civil.DateTime{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := civil.ParseDateTime(encoded)
	if err != nil {
		return err
	}
	dt.DateTime = decoded
	return nil
}

func (dt LocalDateTime) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(dt.String())
}

func (dt *LocalDateTime) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		dt.DateTime = civil.DateTime{}
		return nil
	}

	decoded, err := civil.ParseDateTime(*encoded)
	if err != nil {
		return err
	}
	dt.DateTime = decoded
	return nil
}

// UUID is the shared skel UUID type.
// It is encoded as a UUID string in both JSON and CBOR.
type UUID struct {
	uuid.UUID
}

func NewUUID(id uuid.UUID) UUID {
	return UUID{UUID: id}
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		u.UUID = uuid.UUID{}
		return nil
	}

	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}

	decoded, err := uuid.Parse(encoded)
	if err != nil {
		return err
	}
	u.UUID = decoded
	return nil
}

func (u UUID) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(u.String())
}

func (u *UUID) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		u.UUID = uuid.UUID{}
		return nil
	}

	decoded, err := uuid.Parse(*encoded)
	if err != nil {
		return err
	}
	u.UUID = decoded
	return nil
}

// JSON is the shared skel JSON text type.
// It is encoded as a JSON string in both JSON and CBOR.
type JSON string

func (j JSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(j))
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*j = ""
		return nil
	}
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	*j = JSON(encoded)
	return nil
}

func (j JSON) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(string(j))
}

func (j *JSON) UnmarshalCBOR(data []byte) error {
	var encoded *string
	if err := cbor.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if encoded == nil {
		*j = ""
		return nil
	}
	*j = JSON(*encoded)
	return nil
}
