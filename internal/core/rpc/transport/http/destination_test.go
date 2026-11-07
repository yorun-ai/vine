package http

import (
	"net/http"
	"testing"
)

func TestConsumeDestination(t *testing.T) {
	for _, tc := range []struct {
		input, destination, remaining string
		invalid                       bool
	}{
		{input: "timeout=10s,destination=order.app", destination: "order.app", remaining: "timeout=10s"},
		{input: "destination=order.app,timeout=10s", destination: "order.app", remaining: "timeout=10s"},
		{input: "destination=order.app", destination: "order.app"},
		{input: "timeout=10s", remaining: "timeout=10s"},
		{},
		{input: "destination=", invalid: true},
		{input: "destination=a,destination=b", invalid: true},
		{input: "destination=a,timeout=bad", destination: "a", remaining: "timeout=bad"},
		{input: "destination=a,unknown=b", destination: "a", remaining: "unknown=b"},
		{input: "unknown=b,timeout=10s", remaining: "unknown=b,timeout=10s"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			header := http.Header{}
			if tc.input != "" {
				header.Set(HeaderRpcOptions, tc.input)
			}
			destination, err := ConsumeDestinationFromHeader(header)
			if (err != nil) != tc.invalid {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.invalid {
				if header.Get(HeaderRpcOptions) != tc.input {
					t.Fatal("invalid options mutated")
				}
				return
			}
			if destination != tc.destination || header.Get(HeaderRpcOptions) != tc.remaining {
				t.Fatalf("got %q, %v", destination, header)
			}
		})
	}
	header := http.Header{}
	header.Add(HeaderRpcOptions, "destination=a")
	header.Add(HeaderRpcOptions, "timeout=10s")
	if _, err := ConsumeDestinationFromHeader(header); err == nil {
		t.Fatal("duplicate header accepted")
	}
}
