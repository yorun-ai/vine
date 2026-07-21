package webgw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.yorun.ai/vine/internal/core/meta"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
)

func TestEnsureWebTraceCreatesMissingTraceHeaders(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)

	trace, err := ensureWebTrace(request)
	if err != nil {
		t.Fatalf("ensureWebTrace() error = %v", err)
	}

	got, err := webspec.DecodeTraceFromHeader(request.Header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if trace.Id() != got.Id() || trace.Span() != got.Span() {
		t.Fatalf("unexpected trace id: %s", trace.Id())
	}
	if trace.ParentSpan() == "" {
		t.Fatalf("expected gateway trace parent span")
	}
}

func TestEnsureWebTraceCreatesGatewayTraceFromExistingTraceHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=123e4567e89b12d3a456426614174000,span=1234567890abcdef")

	trace, err := ensureWebTrace(request)
	if err != nil {
		t.Fatalf("ensureWebTrace() error = %v", err)
	}

	if trace.Id() != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("unexpected trace: %s/%s", trace.Id(), trace.Span())
	}
	if trace.ParentSpan() != "1234567890abcdef" {
		t.Fatalf("unexpected parent span: %s", trace.ParentSpan())
	}
	if trace.Span() == "1234567890abcdef" {
		t.Fatalf("expected gateway span to differ from incoming span")
	}
	got, err := webspec.DecodeTraceFromHeader(request.Header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if got.Id() != trace.Id() || got.Span() != trace.Span() {
		t.Fatalf("unexpected web trace header")
	}
}

func TestEnsureWebTraceGeneratesMissingTraceFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=123e4567e89b12d3a456426614174000")

	trace, err := ensureWebTrace(request)
	if err != nil {
		t.Fatalf("ensureWebTrace() error = %v", err)
	}
	if trace.Id() != "123e4567e89b12d3a456426614174000" {
		t.Fatalf("unexpected trace id: %s", trace.Id())
	}
	if !meta.IsValidSpan(trace.Span()) {
		t.Fatalf("expected generated span, got %s", trace.Span())
	}
}

func TestEnsureWebTraceRejectsMissingTraceId(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)
	request.Header.Set(webspec.HeaderWebTrace, "span=1234567890abcdef")

	if _, err := ensureWebTrace(request); err == nil {
		t.Fatal("expected missing trace id to be rejected")
	}
}

func TestEnsureWebInitiatorCreatesInitiatorHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)
	request.Header.Set("User-Agent", "web-test")
	request.Header.Set(webspec.HeaderWebInitiator, "existing-initiator")

	got, err := ensureWebInitiator(request, "192.0.2.1")
	if err != nil {
		t.Fatalf("ensureWebInitiator() error = %v", err)
	}

	initiator, err := meta.DecodeInitiatorFromBase64(request.Header.Get(webspec.HeaderWebInitiator))
	if err != nil {
		t.Fatalf("DecodeInitiatorFromBase64() error = %v", err)
	}
	if initiator.Name() != defaultWebInitiatorName ||
		initiator.Version() != defaultWebInitiatorVersion ||
		initiator.InstanceId() != defaultWebInitiatorInstanceId {
		t.Fatalf("unexpected initiator app")
	}
	if initiator.Dialer() != "web-test" {
		t.Fatalf("unexpected initiator dialer: %s", initiator.Dialer())
	}
	if initiator.IpAddr() != "192.0.2.1" {
		t.Fatalf("unexpected initiator ip: %s", initiator.IpAddr())
	}
	if got.Name() != initiator.Name() || got.IpAddr() != initiator.IpAddr() {
		t.Fatalf("unexpected operation initiator")
	}
	if request.Header.Get(webspec.HeaderWebActor) != "" {
		t.Fatalf("unexpected actor header: %s", request.Header.Get(webspec.HeaderWebActor))
	}
}

func TestEnsureWebTraceRejectsInvalidTrace(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)
	request.Header.Set(webspec.HeaderWebTrace, "id=bad-id,span=1234567890abcdef")

	if _, err := ensureWebTrace(request); err == nil {
		t.Fatal("expected invalid trace to be rejected")
	}
}

func TestEnsureWebInitiatorRejectsInvalidInitiatorIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/hello", nil)

	if _, err := ensureWebInitiator(request, "bad-ip"); err == nil {
		t.Fatal("expected invalid ip to be rejected")
	}
}
