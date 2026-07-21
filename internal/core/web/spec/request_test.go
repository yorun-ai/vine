package spec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.yorun.ai/vine/core/meta"
)

func TestTraceHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	trace := meta.InitialTrace()

	EncodeTraceToHeader(header, trace)
	got, err := DecodeTraceFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if got.Id() != trace.Id() || got.Span() != trace.Span() {
		t.Fatalf("unexpected trace: got=(%s,%s) want=(%s,%s)", got.Id(), got.Span(), trace.Id(), trace.Span())
	}
}

func TestOptionsHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	EncodeOptionsToHeader(header, &Options{Timeout: time.Second})
	if got := header.Get(HeaderWebOptions); got != "timeout=1s" {
		t.Fatalf("unexpected options header: %s", got)
	}

	got, err := DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	if got.Timeout != time.Second {
		t.Fatalf("unexpected timeout: %s", got.Timeout)
	}
}

func TestDecodeOptionsFromHeaderDefaultsToEmpty(t *testing.T) {
	got, err := DecodeOptionsFromHeader(http.Header{})
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	if got.Timeout != 0 {
		t.Fatalf("unexpected timeout: %s", got.Timeout)
	}
}

func TestDecodeOptionsFromHeaderRejectsInvalidValue(t *testing.T) {
	for _, value := range []string{
		"timeout=bad",
		"timeout=0s",
		"timeout=-1s",
		"deadline=1s",
		"timeout=1s,debug=true",
	} {
		header := http.Header{}
		header.Set(HeaderWebOptions, value)
		if _, err := DecodeOptionsFromHeader(header); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestEncodeRequestOptionsToHeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	header := http.Header{}

	EncodeRequestOptionsToHeader(header, ctx)

	got, err := DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	if got.Timeout <= 0 || got.Timeout > time.Second {
		t.Fatalf("unexpected timeout: %s", got.Timeout)
	}
}

func TestDecodeActorFromHeaderRejectsMissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, err := decodeActorFromHeader(req.Header); err == nil {
		t.Fatalf("expected error")
	}
}

func TestInitiatorHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	initiator, err := meta.NewInitiator("portal.app", "1.2.3", "123e4567-e89b-12d3-a456-426614174000", "https", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}

	encodeInitiatorToHeader(header, initiator)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header = header

	got, err := decodeInitiatorFromHeader(req.Header)
	if err != nil {
		t.Fatalf("decodeInitiatorFromHeader() error = %v", err)
	}
	if got == nil || got.Name() != initiator.Name() || got.Dialer() != initiator.Dialer() {
		t.Fatalf("unexpected initiator: %#v", got)
	}
}

func TestDecodeInitiatorFromHeaderRejectsMissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, err := decodeInitiatorFromHeader(req.Header); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeTraceRejectsInvalidPortalTrace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderWebTrace, "id=invalid,span="+meta.InitialTrace().Span())

	if _, err := DecodeTraceFromHeader(req.Header); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeTraceRejectsMissingPortalTraceSpan(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderWebTrace, "id="+meta.InitialTrace().Id())

	if _, err := DecodeTraceFromHeader(req.Header); err == nil {
		t.Fatalf("expected error")
	}
}
