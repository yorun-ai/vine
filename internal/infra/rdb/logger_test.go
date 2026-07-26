package rdb

import "testing"

func TestGetLoggerReusesFallback(t *testing.T) {
	if getLogger(nil) != getLogger(nil) {
		t.Fatal("expected shared fallback logger")
	}
	if getLogger(nil).Name() != "vine:infra:rdb" {
		t.Fatalf("unexpected fallback logger name: %s", getLogger(nil).Name())
	}
}
