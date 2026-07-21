package inproc

import (
	"context"
	"errors"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

func RoundTrip(endpoint string, rpcRequest spec.Request) (spec.Response, ex.Error) {
	handler, ok := getHandler(endpoint)
	if !ok {
		return nil, ex.New(ex.InvocationFailed, "handler is not registered for endpoint "+endpoint)
	}
	if err := contextError(rpcRequest.Context()); err != nil {
		return nil, err
	}

	responseCh := make(chan spec.Response, 1)
	// Match network transport semantics: client-side timeout/cancellation returns
	// without waiting for the server handler to finish.
	go func() {
		responseCh <- handler.ServeRpc(rpcRequest)
	}()

	select {
	case <-rpcRequest.Context().Done():
		return nil, contextError(rpcRequest.Context())
	case rpcResponse := <-responseCh:
		if rpcResponse == nil {
			return nil, ex.New(ex.InvocationFailed, "response is nil")
		}
		return rpcResponse, nil
	}
}

func contextError(ctx context.Context) ex.Error {
	err := ctx.Err()
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return ex.New(ex.InvocationCancelled, err.Error())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ex.New(ex.InvocationTimeout, err.Error())
	}
	return ex.New(ex.InvocationFailed, err.Error())
}
