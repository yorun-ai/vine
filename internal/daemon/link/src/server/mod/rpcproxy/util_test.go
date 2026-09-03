package rpcproxy

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

type _RpcRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f _RpcRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
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

func TestForwardWithTransportPreservesOriginalResponseBody(t *testing.T) {
	originalBody := &closeTrackingBody{Reader: strings.NewReader("ok")}
	transport := _RpcRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       originalBody,
		}, nil
	})
	request, err := http.NewRequest(http.MethodPost, "http://target.local/demo.Service/Invoke", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	response, body, exErr := new(RpcProxy).forwardWithTransport(request.Context(), request, transport)
	if exErr != nil {
		t.Fatalf("forwardWithTransport() error = %v", exErr)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
	if response.Body != originalBody {
		t.Fatal("response body was replaced")
	}
	if originalBody.closed {
		t.Fatal("response body closed before ownership was returned to caller")
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("response Body.Close() error = %v", err)
	}
	if !originalBody.closed {
		t.Fatal("original response body was not closed by caller")
	}
}

func TestForwardWithTransportRejectsOversizedResponseAndClosesBody(t *testing.T) {
	originalBody := &closeTrackingBody{Reader: strings.NewReader("ignored")}
	transport := _RpcRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{},
			Body:          originalBody,
			ContentLength: rpchttp.MaxResponseBodyBytes + 1,
		}, nil
	})
	request, err := http.NewRequest(http.MethodPost, "http://target.local/demo.Service/Invoke", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	_, _, exErr := new(RpcProxy).forwardWithTransport(request.Context(), request, transport)
	if exErr == nil || exErr.Code() != ex.ServiceUnavailable {
		t.Fatalf("forwardWithTransport() error = %v", exErr)
	}
	if !originalBody.closed {
		t.Fatal("oversized response body was not closed")
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
