package skel

import (
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	"github.com/shopspring/decimal"
	clientskel "go.yorun.ai/vrpc/skel"
)

// Sensitive marks complete payloads for redaction.
type Sensitive = clientskel.Sensitive

// PermissionCode is retained for older generated contracts.
// Deprecated: Use string instead.
type PermissionCode string

type Decimal = clientskel.Decimal

type Binary = clientskel.Binary

type Timestamp = clientskel.Timestamp

type Duration = clientskel.Duration

type LocalDate = clientskel.LocalDate

type LocalTime = clientskel.LocalTime

type LocalDateTime = clientskel.LocalDateTime

type UUID = clientskel.UUID

type JSON = clientskel.JSON

func NewDecimal(value decimal.Decimal) Decimal {
	return clientskel.NewDecimal(value)
}

func NewUUID(id uuid.UUID) UUID {
	return clientskel.NewUUID(id)
}

func NewTimestamp(t time.Time) Timestamp {
	return clientskel.NewTimestamp(t)
}

func NewTimestampNow() Timestamp {
	return clientskel.NewTimestampNow()
}

func NewDuration(d time.Duration) Duration {
	return clientskel.NewDuration(d)
}

func NewLocalDate(value civil.Date) LocalDate {
	return clientskel.NewLocalDate(value)
}

func NewLocalDateOf(t time.Time) LocalDate {
	return clientskel.NewLocalDateOf(t)
}

func NewLocalTime(value civil.Time) LocalTime {
	return clientskel.NewLocalTime(value)
}

func NewLocalTimeOf(t time.Time) LocalTime {
	return clientskel.NewLocalTimeOf(t)
}

func NewLocalDateTime(value civil.DateTime) LocalDateTime {
	return clientskel.NewLocalDateTime(value)
}

func NewLocalDateTimeOf(t time.Time) LocalDateTime {
	return clientskel.NewLocalDateTimeOf(t)
}
