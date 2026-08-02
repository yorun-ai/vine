package rpcproxy

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type _RpcRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f _RpcRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRoundTripUsesConfiguredTransport(t *testing.T) {
	proxy := newTestRpcProxy(t, nil)
	clientApp := mustMetaApp(t, "client.app", "11111111-1111-1111-1111-111111111111")
	methodInfo := ensureTestInboundMethodInfo()
	trace := meta.InitialTrace()
	rpcRequest := &spec.RequestImpl{
		ContextValue:    t.Context(),
		TraceValue:      trace,
		ActorValue:      meta.NewAbsentActor(),
		ClientValue:     clientApp,
		MethodInfoValue: methodInfo,
	}

	expectedErr := errors.New("configured transport called")
	transportCalled := false
	proxy.transport = _RpcRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		transportCalled = true
		if !strings.HasSuffix(request.URL.Path, methodInfo.FullURLPath()) {
			t.Fatalf("unexpected Rpc request path: %s", request.URL.Path)
		}
		return nil, expectedErr
	})

	_, exErr := proxy.roundTrip("http://target.test/rpc/invoke", rpcRequest)

	if !transportCalled {
		t.Fatal("expected configured transport to be called")
	}
	if exErr == nil || !strings.Contains(exErr.Message(), expectedErr.Error()) {
		t.Fatalf("unexpected Rpc error: %v", exErr)
	}
}

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
