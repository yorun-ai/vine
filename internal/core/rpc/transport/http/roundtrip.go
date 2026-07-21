package http

import (
	"context"
	"errors"
	"net"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/util/httputil"
)

var defaultHTTPClient = httputil.NewH2CClient()

func RoundTrip(endpoint string, rpcRequest spec.Request) (spec.Response, ex.Error) {
	httpRequest, err := encodeRequest(endpoint, rpcRequest)
	if err != nil {
		return nil, ex.New(ex.InvocationFailed, err.Error())
	}

	httpResponse, err := defaultHTTPClient.Do(httpRequest)
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
