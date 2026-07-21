package rpcproxy

import (
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
)

func TestMapGatewayResponseErrorMapsUnresponsiveCodes(t *testing.T) {
	tests := []struct {
		name string
		err  ex.Error
		want ex.Code
	}{
		{
			name: "cancelled",
			err:  ex.New(ex.InvocationCancelled, "context canceled"),
			want: ex.ServiceUnavailable,
		},
		{
			name: "timeout",
			err:  ex.New(ex.InvocationTimeout, "context deadline exceeded"),
			want: ex.GatewayTimeout,
		},
		{
			name: "responsive",
			err:  ex.New(ex.InvalidRequest, "bad request"),
			want: ex.InvalidRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapGatewayResponseError(tc.err)
			if got.Code() != tc.want {
				t.Fatalf("unexpected code: got %s want %s", got.Code(), tc.want)
			}
			if got.Message() != tc.err.Message() {
				t.Fatalf("unexpected message: got %q want %q", got.Message(), tc.err.Message())
			}
		})
	}
}

func TestMapGatewayResponseErrorPreservesResponsiveError(t *testing.T) {
	err := ex.New(ex.NotFound, "missing", ex.WithReason("not_found"), ex.WithDetail("detail"))

	got := mapGatewayResponseError(err)
	if got != err {
		t.Fatal("expected responsive error to be preserved")
	}
}
