package http

import (
	"context"
	"errors"
	"net"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/util/httputil"
)

var defaultHTTPClient = httputil.NewH2CClient()

func RoundTrip(endpoint string, rpcRequest spec.Request) (spec.Response, ex.Error) {
	return RoundTripWithPrepared(endpoint, rpcRequest, nil)
}

func RoundTripWithPrepared(endpoint string, rpcRequest spec.Request, prepared func()) (spec.Response, ex.Error) {
	return roundTrip(endpoint, rpcRequest, prepared, defaultHTTPClient.Do)
}

// RoundTripWithTransport sends an Rpc request through the provided HTTP
// transport. It allows proxy layers to use the same transport configuration
// for both decoded HTTP requests and Rpc requests received through inproc.
func RoundTripWithTransport(endpoint string, rpcRequest spec.Request, transport http.RoundTripper) (spec.Response, ex.Error) {
	return roundTrip(endpoint, rpcRequest, nil, transport.RoundTrip)
}

func roundTrip(
	endpoint string,
	rpcRequest spec.Request,
	prepared func(),
	do func(*http.Request) (*http.Response, error),
) (spec.Response, ex.Error) {
	httpRequest, err := encodeRequest(endpoint, rpcRequest)
	if err != nil {
		return nil, ex.New(ex.InvocationFailed, err.Error())
	}
	if prepared != nil {
		prepared()
	}

	httpResponse, err := do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, ex.New(ex.InvocationCancelled, err.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ex.New(ex.InvocationTimeout, err.Error())
		}
		if _, ok := errors.AsType[*net.DNSError](err); ok {
			return nil, ex.New(ex.ServerUnreachable, err.Error())
		}
		if _, ok := errors.AsType[*net.OpError](err); ok {
			return nil, ex.New(ex.ServerUnreachable, err.Error())
		}
		return nil, ex.New(ex.InvocationFailed, err.Error())
	}
	defer func() { _ = httpResponse.Body.Close() }()

	rpcResponse, err := decodeResponse(httpResponse, rpcRequest.MethodInfo())
	if err != nil {
		return nil, ex.New(ex.InvocationFailed, err.Error())
	}
	return rpcResponse, nil
}
