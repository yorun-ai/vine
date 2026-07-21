package meta

import "testing"

func TestIsValidIdRejectsNonHexValue(t *testing.T) {
	id := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	if IsValidId(id) {
		t.Fatalf("expected invalid id: %s", id)
	}
}

func TestIsValidIdRejectsAllZeroTraceID(t *testing.T) {
	if IsValidId("00000000000000000000000000000000") {
		t.Fatalf("expected all-zero trace id to be invalid")
	}
}

func TestIsValidSpan(t *testing.T) {
	if !IsValidSpan("4bf92f3577b34da6") {
		t.Fatalf("expected valid span")
	}
	if IsValidSpan("zzzzzzzzzzzzzzzz") {
		t.Fatalf("expected invalid non-hex span")
	}
	if IsValidSpan("0000000000000000") {
		t.Fatalf("expected all-zero span id to be invalid")
	}
}

func TestTraceNewChildTrace(t *testing.T) {
	parent := InitialTrace()
	child := parent.NewChildTrace()
	if child.Id() != parent.Id() {
		t.Fatalf("unexpected child trace id: got %s want %s", child.Id(), parent.Id())
	}
	if child.ParentSpan() != parent.Span() {
		t.Fatalf("unexpected parent span: got %s want %s", child.ParentSpan(), parent.Span())
	}
	if child.Span() == parent.Span() {
		t.Fatalf("expected child span to differ from parent span")
	}
}

func TestTraceDelimitedRoundTrip(t *testing.T) {
	trace, err := NewTrace("123e4567e89b12d3a456426614174000", "1234567890abcdef")
	if err != nil {
		t.Fatalf("NewTrace() error = %v", err)
	}

	encoded := EncodeTraceToDelimited(trace)
	if encoded != "id=123e4567e89b12d3a456426614174000,span=1234567890abcdef" {
		t.Fatalf("unexpected trace string: %s", encoded)
	}
	got, err := DecodeTraceFromDelimited(encoded)
	if err != nil {
		t.Fatalf("DecodeTraceFromDelimited() error = %v", err)
	}
	if got.Id() != trace.Id() || got.Span() != trace.Span() {
		t.Fatalf("unexpected trace: %s/%s", got.Id(), got.Span())
	}
}

func TestDecodeTraceFromDelimitedRejectsInvalidValue(t *testing.T) {
	if _, err := DecodeTraceFromDelimited("id=bad-id,span=1234567890abcdef"); err == nil {
		t.Fatalf("expected invalid trace string to be rejected")
	}
	if _, err := DecodeTraceFromDelimited("id=123e4567e89b12d3a456426614174000"); err == nil {
		t.Fatalf("expected missing span to be rejected")
	}
}

func TestDecodeTraceFromDelimitedOrNewSpan(t *testing.T) {
	got, err := DecodeTraceFromDelimitedOrNewSpan("id=123e4567e89b12d3a456426614174000")
	if err != nil {
		t.Fatalf("DecodeTraceFromDelimitedOrNewSpan() error = %v", err)
	}
	if got.Id() != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("unexpected trace id: %s", got.Id())
	}
	if !IsValidSpan(got.Span()) {
		t.Fatalf("expected generated span, got %s", got.Span())
	}

	got, err = DecodeTraceFromDelimitedOrNewSpan("id=123e4567e89b12d3a456426614174000,span=1234567890abcdef")
	if err != nil {
		t.Fatalf("DecodeTraceFromDelimitedOrNewSpan() error = %v", err)
	}
	if got.Span() != "1234567890abcdef" {
		t.Fatalf("unexpected span: %s", got.Span())
	}
}

func TestDecodeTraceFromDelimitedOrNewSpanRejectsMissingId(t *testing.T) {
	if _, err := DecodeTraceFromDelimitedOrNewSpan("span=1234567890abcdef"); err == nil {
		t.Fatalf("expected missing id to be rejected")
	}
}
