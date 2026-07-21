package spec

import (
	"strings"
	"testing"
)

func TestCheckValueNotNilRejectsNilSlice(t *testing.T) {
	var items []string

	err := CheckValueNotNil(items, "result.items")
	if err == nil {
		t.Fatalf("expected nil slice error")
	}
	if !strings.Contains(err.Error(), "result.items cannot be nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJoinPathSegments(t *testing.T) {
	if got := JoinPath("result", "items"); got != "result.items" {
		t.Fatalf("unexpected field path: %s", got)
	}
	if got := JoinIndex("result.items", 3); got != "result.items[3]" {
		t.Fatalf("unexpected index path: %s", got)
	}
	if got := JoinMapKey("result.attrs", "main"); got != `result.attrs["main"]` {
		t.Fatalf("unexpected map key path: %s", got)
	}
}
