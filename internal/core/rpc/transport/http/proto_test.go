package http

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vcode"
)

func testProtoRequestHeaders(t *testing.T) http.Header {
	t.Helper()

	trace := meta.InitialTrace()
	if trace == nil {
		t.Fatalf("expected initial trace")
	}
	client, err := meta.NewApp("demo", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	header := http.Header{}
	EncodeContentTypeHeadersToHeaderByMethod(header, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	EncodeTraceToHeader(header, trace)
	EncodeClientToHeader(header, client)
	return header
}

func testProtoResponseHeaders(t *testing.T) http.Header {
	t.Helper()

	trace := meta.InitialTrace()
	if trace == nil {
		t.Fatalf("expected initial trace")
	}
	server, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	header := http.Header{}
	EncodeContentTypeHeadersToHeader(header, ContentTypeJson)
	EncodeTraceToHeader(header, trace)
	EncodeStatusCodeToHeader(header, ex.OK)
	EncodeServerToHeader(header, server)
	return header
}

func TestCheckRequestMethod(t *testing.T) {
	req, err := http.NewRequest(RequestMethod, "http://localhost", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if err := CheckRequestMethod(req); err != nil {
		t.Fatalf("CheckRequestMethod() error = %v", err)
	}

	req.Method = http.MethodGet
	if err := CheckRequestMethod(req); err == nil {
		t.Fatalf("expected invalid method to be rejected")
	}
}

func TestCheckRequestHeaders(t *testing.T) {
	header := testProtoRequestHeaders(t)
	header.Set(HeaderAccept, "application/json, "+ContentTypeJson)

	if err := CheckRequestHeaders(header); err != nil {
		t.Fatalf("CheckRequestHeaders() error = %v", err)
	}

	header = testProtoRequestHeaders(t)
	header.Add(HeaderRpcTrace, "duplicated")
	if err := CheckRequestHeaders(header); err == nil {
		t.Fatalf("expected duplicated request header to be rejected")
	}

	header = testProtoRequestHeaders(t)
	header.Set(HeaderContentType, "application/json")
	if err := CheckRequestHeaders(header); err == nil {
		t.Fatalf("expected invalid request content-type to be rejected")
	}

	header = testProtoRequestHeaders(t)
	header.Set(HeaderAccept, "application/json, "+ContentTypeCbor+";q=1, "+ContentTypeJson+";q=0.8")
	header.Set(HeaderContentType, ContentTypeJson+"; charset=utf-8")
	if err := CheckRequestHeaders(header); err != nil {
		t.Fatalf("expected vrpc media type with params to be accepted, got %v", err)
	}
}

func TestCheckRequestContentTypeHeader(t *testing.T) {
	header := http.Header{}
	header.Set(HeaderContentType, ContentTypeCbor+"; charset=utf-8")
	if err := CheckRequestContentTypeHeader(header); err != nil {
		t.Fatalf("CheckRequestContentTypeHeader() error = %v", err)
	}

	header = http.Header{}
	if err := CheckRequestContentTypeHeader(header); err == nil {
		t.Fatalf("expected missing content-type to be rejected")
	}

	header = http.Header{}
	header.Set(HeaderContentType, "application/json")
	if err := CheckRequestContentTypeHeader(header); err == nil {
		t.Fatalf("expected invalid content-type to be rejected")
	}
}

func TestCheckResponseHeaders(t *testing.T) {
	header := testProtoResponseHeaders(t)
	if err := CheckResponseHeaders(header); err != nil {
		t.Fatalf("CheckResponseHeaders() error = %v", err)
	}

	header = testProtoResponseHeaders(t)
	header.Del(HeaderRpcStatus)
	if err := CheckResponseHeaders(header); err == nil {
		t.Fatalf("expected missing response header to be rejected")
	}
}

func TestEncodeFixedRequestAndResponseHeaders(t *testing.T) {
	requestHeader := http.Header{}
	EncodeContentTypeHeadersToHeaderByMethod(requestHeader, newStandaloneMethodInfo(reflect.TypeOf(pingArguments{}), reflect.TypeOf(""), false, false))
	if requestHeader.Get(HeaderAccept) != ContentTypeJson {
		t.Fatalf("unexpected accept header: %s", requestHeader.Get(HeaderAccept))
	}
	if requestHeader.Get(HeaderContentType) != ContentTypeJson {
		t.Fatalf("unexpected request content-type: %s", requestHeader.Get(HeaderContentType))
	}

	responseHeader := http.Header{}
	EncodeContentTypeHeadersToHeader(responseHeader, ContentTypeJson)
	if responseHeader.Get(HeaderContentType) != ContentTypeJson {
		t.Fatalf("unexpected response content-type: %s", responseHeader.Get(HeaderContentType))
	}
}

func TestAcceptsContentTypeIgnoresParameters(t *testing.T) {
	value := "application/json, " + ContentTypeCbor + ";q=1, " + ContentTypeJson + ";q=0.8"
	if !AcceptsContentType(value, ContentTypeCbor) {
		t.Fatalf("expected cbor accept with parameters to be matched")
	}
	if !AcceptsContentType(value, ContentTypeJson) {
		t.Fatalf("expected json accept with parameters to be matched")
	}
}

func TestTraceHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	trace := meta.InitialTrace()
	if trace == nil {
		t.Fatalf("expected initial trace")
	}

	EncodeTraceToHeader(header, trace)
	if got := header.Get(HeaderRpcTrace); got != "id="+trace.Id()+",span="+trace.Span() {
		t.Fatalf("unexpected trace header: %s", got)
	}
	got, err := DecodeTraceFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeTraceFromHeader() error = %v", err)
	}
	if got.Id() != trace.Id() || got.Span() != trace.Span() {
		t.Fatalf("unexpected trace: got=%s/%s want=%s/%s", got.Id(), got.Span(), trace.Id(), trace.Span())
	}
}

func TestOptionsHeaderRoundTrip(t *testing.T) {
	header := http.Header{}

	EncodeOptionsToHeader(header, &Options{Timeout: time.Second})
	if got := header.Get(HeaderRpcOptions); got != "timeout=1s" {
		t.Fatalf("unexpected options header: %s", got)
	}

	got, err := DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	if got.Timeout != time.Second {
		t.Fatalf("unexpected timeout: got %s want %s", got.Timeout, time.Second)
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
	tests := []string{
		"timeout=bad",
		"timeout=0s",
		"deadline=2026-07-16T10:20:30Z",
		"timeout=1s,wait=async",
	}

	for _, value := range tests {
		header := http.Header{}
		header.Set(HeaderRpcOptions, value)
		if _, err := DecodeOptionsFromHeader(header); err == nil {
			t.Fatalf("expected invalid options header to be rejected: %s", value)
		}
	}
}

func TestClientAndServerHeaderRoundTrip(t *testing.T) {
	client, err := meta.NewApp("client", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp(client) error = %v", err)
	}
	server, err := meta.NewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp(server) error = %v", err)
	}
	header := http.Header{}

	EncodeClientToHeader(header, client)
	if got := header.Get(HeaderRpcClient); got != "name=client,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected client header: %s", got)
	}
	gotClient, err := DecodeClientFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeClientFromHeader() error = %v", err)
	}
	if gotClient.Name() != client.Name() || gotClient.Version() != client.Version() || gotClient.InstanceId() != client.InstanceId() {
		t.Fatalf("unexpected client app")
	}

	EncodeServerToHeader(header, server)
	if got := header.Get(HeaderRpcServer); got != "name=server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected server header: %s", got)
	}
	gotServer, err := DecodeServerFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeServerFromHeader() error = %v", err)
	}
	if gotServer.Name() != server.Name() || gotServer.Version() != server.Version() || gotServer.InstanceId() != server.InstanceId() {
		t.Fatalf("unexpected server app")
	}
}

func TestDecodeClientFromHeaderExported(t *testing.T) {
	client, err := meta.NewApp("client", "1.0.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp(client) error = %v", err)
	}
	header := http.Header{}
	EncodeClientToHeader(header, client)

	got, err := DecodeClientFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeClientFromHeader() error = %v", err)
	}
	if got.Name() != client.Name() || got.Version() != client.Version() || got.InstanceId() != client.InstanceId() {
		t.Fatalf("unexpected client app")
	}
}

func TestParseServiceAndMethodFromPath(t *testing.T) {
	serviceName, methodName, err := ParseServiceAndMethodFromPath("/demo.service/Ping")
	if err != nil {
		t.Fatalf("ParseServiceAndMethodFromPath() error = %v", err)
	}
	if serviceName != "demo.service" || methodName != "Ping" {
		t.Fatalf("unexpected path parts: %s %s", serviceName, methodName)
	}
}

func TestActorHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	actor := meta.NewAnonymousActor()

	EncodeActorToHeader(header, actor)
	got, err := DecodeActorFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeActorFromHeader() error = %v", err)
	}
	if got.Type() != actor.Type() {
		t.Fatalf("unexpected actor type: got=%s want=%s", got.Type(), actor.Type())
	}
}

func TestDecodeActorHeaderDefaultsToAbsent(t *testing.T) {
	got, err := DecodeActorFromHeader(http.Header{})
	if err != nil {
		t.Fatalf("DecodeActorFromHeader() error = %v", err)
	}
	if got.Type() != meta.ActorTypeAbsent {
		t.Fatalf("unexpected default actor type: %s", got.Type())
	}
}

func TestInitiatorHeaderRoundTrip(t *testing.T) {
	header := http.Header{}
	initiator, err := meta.NewInitiator(
		"demo.service",
		"1.2.3",
		"123e4567-e89b-12d3-a456-426614174000",
		"http",
		"127.0.0.1",
	)
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}

	EncodeInitiatorToHeader(header, initiator)
	got, err := DecodeInitiatorFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeInitiatorFromHeader() error = %v", err)
	}
	if got == nil {
		t.Fatalf("expected initiator")
	}
	if got.Name() != initiator.Name() || got.Version() != initiator.Version() || got.InstanceId() != initiator.InstanceId() {
		t.Fatalf("unexpected initiator app")
	}
	if got.Dialer() != initiator.Dialer() {
		t.Fatalf("unexpected initiator dialer: got=%s want=%s", got.Dialer(), initiator.Dialer())
	}
	if got.IpAddr() != "127.0.0.1" {
		t.Fatalf("unexpected initiator ip: %v", got.IpAddr())
	}
}

func TestStatusCodeHeaderRoundTrip(t *testing.T) {
	header := http.Header{}

	EncodeStatusCodeToHeader(header, ex.OK)
	got, err := DecodeStatusCodeFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeStatusCodeFromHeader() error = %v", err)
	}
	if got != ex.OK {
		t.Fatalf("unexpected status code: got=%s want=%s", got, ex.OK)
	}
}

func TestErrorPayloadRoundTrip(t *testing.T) {
	errPayload := ex.New(ex.InvalidRequest, "bad request", ex.WithDetail("detail"))

	body := ex.EncodeError(errPayload, vcode.MustMarshalJson)
	got, err := ex.DecodeError(body, json.Unmarshal)
	if err != nil {
		t.Fatalf("DecodeError() error = %v", err)
	}
	if got.Code() != errPayload.Code() || got.Message() != errPayload.Message() || got.Detail() != errPayload.Detail() {
		t.Fatalf("unexpected error payload: got=%s/%s/%s", got.Code(), got.Message(), got.Detail())
	}
}

func TestDecodeServiceAndMethodFromPath(t *testing.T) {
	service, method, err := ParseServiceAndMethodFromPath("/test.TestService/ping")
	if err != nil {
		t.Fatalf("ParseServiceAndMethodFromPath() error = %v", err)
	}
	if service != "test.TestService" || method != "ping" {
		t.Fatalf("unexpected parsed path: service=%s method=%s", service, method)
	}

	if _, _, err := ParseServiceAndMethodFromPath("/bad/path/extra"); err == nil {
		t.Fatalf("expected invalid path to be rejected")
	}
}
