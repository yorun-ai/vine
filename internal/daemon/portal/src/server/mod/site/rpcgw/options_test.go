package rpcgw

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

func TestOptionsDiscardUnknownFields(t *testing.T) {
	for _, value := range []string{"destination=private.app", "timeout=10s,destination=private.app", "future=value", "timeout=10s,future=value,destination=private.app"} {
		request := httptest.NewRequest("POST", "http://portal.test/invoke", nil)
		request.Header.Set(rpchttp.HeaderRpcOptions, value)
		next, cancel, err := requestWithRpcOptionsTimeout(request, context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer cancel()
		options, err := rpchttp.DecodeOptionsFromHeader(next.Header)
		if err != nil {
			t.Fatalf("nonstandard options forwarded: %v", err)
		}
		want := defaultRpcTimeout
		if value[0] == 't' {
			want = 10 * time.Second
		}
		if options.Timeout != want {
			t.Fatalf("timeout = %v, want %v", options.Timeout, want)
		}
		if request.Header.Get(rpchttp.HeaderRpcOptions) != value {
			t.Fatal("original request mutated")
		}
	}
}

func TestOptionsRejectMalformedFields(t *testing.T) {
	for _, value := range []string{"timeout=bad,future=value", "timeout=0s,future=value", "timeout=10s,timeout=20s", "future", ""} {
		request := httptest.NewRequest("POST", "http://portal.test/invoke", nil)
		request.Header.Set(rpchttp.HeaderRpcOptions, value)
		_, cancel, err := requestWithRpcOptionsTimeout(request, context.Background())
		if cancel != nil {
			cancel()
		}
		if err == nil {
			t.Errorf("accepted invalid options %q", value)
		}
	}
}
