package ex

import "testing"

func TestInvalidCodeMetadata(t *testing.T) {
	code := Code("BOGUS")

	if code.IsValid() {
		t.Fatalf("expected invalid code")
	}
	if got := code.Type(); got != InvalidType {
		t.Fatalf("unexpected invalid type: got %s", got)
	}
	if got := code.Category(); got != InvalidCategory {
		t.Fatalf("unexpected invalid category: got %s", got)
	}
	if code.IsUnresponsive() {
		t.Fatalf("invalid code should not be unresponsive")
	}
	if code.CanRaiseDirectly() {
		t.Fatalf("invalid code should not be directly raisable")
	}
	if got := code.DefaultMessage(); got != "" {
		t.Fatalf("unexpected default message for invalid code: %q", got)
	}
}

func TestParseCode(t *testing.T) {
	if _, err := ParseCode(string(OK)); err != nil {
		t.Fatalf("expected OK to parse, got error: %v", err)
	}
	if _, err := ParseCode("BOGUS"); err == nil {
		t.Fatalf("expected invalid code parse to fail")
	}
}
