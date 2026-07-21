package rdb

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUUIDV7String(t *testing.T) {
	value := NewUUIDV7String()
	if err := uuid.Validate(value); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	parsed := uuid.MustParse(value)
	if parsed.Version() != 7 {
		t.Fatalf("expected v7 uuid, got v%d", parsed.Version())
	}
}
