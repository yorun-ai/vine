package httputil

import (
	"testing"

	"golang.org/x/net/http2"
)

func TestNewH2CClient(t *testing.T) {
	client := NewH2CClient()
	transport, ok := client.Transport.(*http2.Transport)
	if !ok {
		t.Fatalf("unexpected transport type: %T", client.Transport)
	}
	if !transport.AllowHTTP {
		t.Fatal("expected h2c client to allow plain HTTP")
	}
	if !transport.DisableCompression {
		t.Fatal("expected h2c client to disable compression")
	}
}
