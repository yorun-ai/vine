package rpcgw

import (
	"context"
	"fmt"
	"net/http"
	"time"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
)

const (
	defaultRpcTimeout = 30 * time.Second
	maxRpcTimeout     = 120 * time.Second
)

func requestWithRpcOptionsTimeout(request *http.Request, gatewayContext context.Context) (*http.Request, context.CancelFunc, error) {
	options, err := rpchttp.DecodeOptionsFromHeader(request.Header)
	if err != nil {
		return request, nil, err
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultRpcTimeout
	}
	if timeout > maxRpcTimeout {
		return request, nil, fmt.Errorf("invalid request header %s: timeout exceeds %s", rpchttp.HeaderRpcOptions, maxRpcTimeout)
	}

	ctx, cancel := gwutil.ContextWithoutClientCancel(request.Context(), gatewayContext, timeout)
	next := request.Clone(ctx)
	next.Header = request.Header.Clone()
	rpchttp.EncodeOptionsToHeader(next.Header, &rpchttp.Options{Timeout: timeout})
	return next, cancel, nil
}
