package http

import (
	"net/http"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vcode"
)

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
	if gotServer.Name() != "server" || gotServer.Version() != server.Version() || gotServer.InstanceId() != server.InstanceId() {
		t.Fatalf("unexpected server app")
	}
}

func TestEncodeClientToHeaderNormalizesGoVersionPrefix(t *testing.T) {
	client, err := meta.NewApp("vine.hub", "v0.17.0", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	header := http.Header{}

	EncodeClientToHeader(header, client)
	if got := header.Get(HeaderRpcClient); got != "name=vine.hub,version=0.17.0,instanceId=123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected client header: %s", got)
	}
	got, err := DecodeClientFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeClientFromHeader() error = %v", err)
	}
	if got.Version() != "0.17.0" {
		t.Fatalf("unexpected version: %s", got.Version())
	}
}

func TestDecodeClientFromHeaderAcceptsGoVersionPrefix(t *testing.T) {
	header := http.Header{}
	header.Set(HeaderRpcClient, "name=vine.hub,version=v0.15.8,instanceId=123e4567-e89b-12d3-a456-426614174000")

	got, err := DecodeClientFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeClientFromHeader() error = %v", err)
	}
	if got.Name() != "vine.hub" || got.Version() != "v0.15.8" {
		t.Fatalf("unexpected client app: %s %s", got.Name(), got.Version())
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
	got, err := ex.DecodeError(body, unmarshalJson)
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
