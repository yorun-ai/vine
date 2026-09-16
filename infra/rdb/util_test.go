package rdb

import (
	"testing"

	"uuid"
)

func TestNewUUIDV7String(t *testing.T) {
	value := NewUUIDV7String()
	parsed, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if version := parsed[6] >> 4; version != 7 {
		t.Fatalf("expected v7 uuid, got v%d", version)
	}
}
