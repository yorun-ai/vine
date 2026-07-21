package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

func TestDecodeResponseDecodesErrorPayload(t *testing.T) {
	method := testServiceInfo().Methods()[0]
	wantErr := ex.New(ex.NotFound, "missing user")
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ErrorValue:  wantErr,
	}

	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, nil, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}
	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()

	got, err := decodeResponse(httpResp, method)
	if err != nil {
		t.Fatalf("decodeResponse() error = %v", err)
	}
	if got.Error() == nil {
		t.Fatalf("expected response error")
	}
	if got.Error().Code() != wantErr.Code() {
		t.Fatalf("unexpected error code: got %s want %s", got.Error().Code(), wantErr.Code())
	}
	if got.Error().Message() != wantErr.Message() {
		t.Fatalf("unexpected error message: got %q want %q", got.Error().Message(), wantErr.Message())
	}
}

func TestDecodeResponseRejectsMissingBody(t *testing.T) {
	method := testServiceInfo().Methods()[0]
	recorder := httptest.NewRecorder()
	header := recorder.Header()
	EncodeContentTypeHeadersToHeader(header, ContentTypeJson)
	EncodeStatusCodeToHeader(header, ex.OK)
	EncodeServerToHeader(header, testServerApp())
	recorder.WriteHeader(ResponseStatusCode)

	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()

	_, err := decodeResponse(httpResp, method)
	if err == nil || err.Error() != "missing response body" {
		t.Fatalf("expected missing response body error, got %v", err)
	}
}

func TestWriteResponseWrapsSuccessResultAndError(t *testing.T) {
	method := testServiceInfo().Methods()[0]
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ResultValue: "pong",
		ErrorValue:  ex.NewOK(),
	}

	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, nil, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}
	if got := recorder.Header().Get(HeaderRpcTrace); got != "" {
		t.Fatalf("unexpected response trace header: %s", got)
	}

	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(bodyBytes); got != `{"result":"pong","error":null}` {
		t.Fatalf("unexpected response body: %s", got)
	}
}

func TestWriteResponseForRequestUsesCborWhenAcceptedAndBinaryResultType(t *testing.T) {
	method := newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, true)
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ResultValue: "pong",
		ErrorValue:  ex.NewOK(),
	}
	req := httptest.NewRequest(RequestMethod, "http://localhost:8080/test.standalone/ping", nil)
	req.Header.Set(HeaderAccept, ContentTypeCbor)

	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, req, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.Header.Get(HeaderContentType) != ContentTypeCbor {
		t.Fatalf("unexpected response content-type: %s", httpResp.Header.Get(HeaderContentType))
	}

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	got, err := cbor.Diagnose(bodyBytes)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got != `{`+`"result": "pong", "error": null`+`}` {
		t.Fatalf("unexpected cbor body: %s", got)
	}

	httpResp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	decoded, err := decodeResponse(httpResp, method)
	if err != nil {
		t.Fatalf("decodeResponse() error = %v", err)
	}
	if decoded.Result() != "pong" {
		t.Fatalf("unexpected result: %#v", decoded.Result())
	}
}

func TestWriteResponseForRequestPrefersCborWhenAcceptIncludesJsonAndCbor(t *testing.T) {
	method := newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, true)
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ResultValue: "pong",
		ErrorValue:  ex.NewOK(),
	}
	req := httptest.NewRequest(RequestMethod, "http://localhost:8080/test.standalone/ping", nil)
	req.Header.Set(HeaderAccept, ContentTypeJson+", "+ContentTypeCbor)

	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, req, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	httpResp := recorder.Result()
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.Header.Get(HeaderContentType) != ContentTypeCbor {
		t.Fatalf("expected cbor response when accept includes both json and cbor, got %s", httpResp.Header.Get(HeaderContentType))
	}
}

func TestClearResponseErrorDetailClearsJsonDetail(t *testing.T) {
	method := testServiceInfo().Methods()[0]
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ErrorValue:  ex.New(ex.OperationFailed, "write failed", ex.WithReason("quota-exceeded"), ex.WithDetail("disk offline")),
	}
	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, nil, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	gotBody, err := ClearResponseErrorDetail(recorder.Body.Bytes(), ContentTypeJson)
	if err != nil {
		t.Fatalf("ClearResponseErrorDetail() error = %v", err)
	}
	responsePayload := &_ResponsePayloadJson{}
	if err := json.Unmarshal(gotBody, responsePayload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	errorPayload := map[string]any{}
	if err := json.Unmarshal(responsePayload.Error, &errorPayload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, ok := errorPayload["detail"]; !ok || got != "" {
		t.Fatalf("expected empty detail field, got %#v", errorPayload)
	}
	gotErr, err := ex.DecodeError(responsePayload.Error, json.Unmarshal)
	if err != nil {
		t.Fatalf("DecodeError() error = %v", err)
	}
	if gotErr.Reason() != "quota-exceeded" || gotErr.Detail() != "" {
		t.Fatalf("unexpected error payload: reason=%q detail=%q", gotErr.Reason(), gotErr.Detail())
	}
}

func TestClearResponseErrorDetailClearsCborDetail(t *testing.T) {
	method := newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, true)
	msg := &spec.ResponseImpl{
		ServerValue: testServerApp(),
		MethodValue: method,
		ErrorValue:  ex.New(ex.OperationFailed, "write failed", ex.WithReason("quota-exceeded"), ex.WithDetail("disk offline")),
	}
	req := httptest.NewRequest(RequestMethod, "http://localhost:8080/test.standalone/ping", nil)
	req.Header.Set(HeaderAccept, ContentTypeCbor)
	recorder := httptest.NewRecorder()
	if err := WriteResponse(recorder, req, msg); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	gotBody, err := ClearResponseErrorDetail(recorder.Body.Bytes(), ContentTypeCbor)
	if err != nil {
		t.Fatalf("ClearResponseErrorDetail() error = %v", err)
	}
	responsePayload := &_ResponsePayloadCbor{}
	if err := cbor.Unmarshal(gotBody, responsePayload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	gotErr, err := ex.DecodeError(responsePayload.Error, cbor.Unmarshal)
	if err != nil {
		t.Fatalf("DecodeError() error = %v", err)
	}
	if gotErr.Reason() != "quota-exceeded" || gotErr.Detail() != "" {
		t.Fatalf("unexpected error payload: reason=%q detail=%q", gotErr.Reason(), gotErr.Detail())
	}
}
