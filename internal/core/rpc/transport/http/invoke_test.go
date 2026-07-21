package http

import (
	"context"
	"io"
	"testing"

	"go.yorun.ai/vine/internal/core/meta"
)

func TestBuildInvokeRequest(t *testing.T) {
	rpcCtx := testContext()
	initiator := testInitiator(t)
	request := BuildInvokeRequest(InvokeRequest{
		Context:         context.Background(),
		Endpoint:        "https://portal-access.local/",
		ServiceSkelName: "demo.UserService",
		MethodSkelName:  "get",
		Params:          map[string]any{"id": "u-1"},
		Trace:           rpcCtx.Trace(),
		Client:          rpcCtx.Client(),
		Actor:           rpcCtx.Actor(),
		Initiator:       initiator,
	})

	if request.Method != RequestMethod {
		t.Fatalf("unexpected method: %s", request.Method)
	}
	if request.URL.String() != "https://portal-access.local/demo.UserService/get" {
		t.Fatalf("unexpected url: %s", request.URL.String())
	}
	if request.Header.Get(HeaderAccept) != ContentTypeJson {
		t.Fatalf("unexpected accept: %s", request.Header.Get(HeaderAccept))
	}
	if request.Header.Get(HeaderContentType) != ContentTypeJson {
		t.Fatalf("unexpected content-type: %s", request.Header.Get(HeaderContentType))
	}
	if request.Header.Get(HeaderRpcTrace) != "id="+rpcCtx.Trace().Id()+",span="+rpcCtx.Trace().Span() {
		t.Fatalf("unexpected trace header: %s", request.Header.Get(HeaderRpcTrace))
	}
	if request.Header.Get(HeaderRpcClient) == "" {
		t.Fatalf("expected client header")
	}
	if request.Header.Get(HeaderRpcActor) == "" {
		t.Fatalf("expected actor header")
	}
	if request.Header.Get(HeaderRpcInitiator) == "" {
		t.Fatalf("expected initiator header")
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != `{"params":{"id":"u-1"}}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func testInitiator(t *testing.T) meta.Initiator {
	t.Helper()

	initiator, err := meta.NewInitiator("demo.client", "1.0.0", "123e4567-e89b-12d3-a456-426614174002", "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}
	return initiator
}
