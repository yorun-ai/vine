package rdb

import "testing"

func TestGetLoggerReusesFallback(t *testing.T) {
	first := getLogger(t.Context())
	second := getLogger(t.Context())
	if first != second {
		t.Fatal("expected shared fallback logger")
	}
	if first.Name() != "vine:infra:rdb" {
		t.Fatalf("unexpected fallback logger name: %s", first.Name())
	}
}
