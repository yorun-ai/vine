package rpcgw

import (
	"context"
	"fmt"
	"net/http"
	"time"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

const (
	defaultRpcTimeout = 30 * time.Second
	maxRpcTimeout     = 120 * time.Second
)

func requestWithRpcOptionsTimeout(request *http.Request, gatewayContext context.Context) (*http.Request, context.CancelFunc, error) {
	header := request.Header.Clone()
	values := header.Values(rpchttp.HeaderRpcOptions)
	if len(values) > 1 {
		return request, nil, fmt.Errorf("invalid request header %s", rpchttp.HeaderRpcOptions)
	}
	if len(values) == 1 {
		fields, err := vrpchttp.DecodeFields(values[0])
		if err != nil {
			return request, nil, fmt.Errorf("invalid request header %s", rpchttp.HeaderRpcOptions)
		}
		header.Del(rpchttp.HeaderRpcOptions)
		if timeout := fields["timeout"]; timeout != "" {
			header.Set(rpchttp.HeaderRpcOptions, vrpchttp.EncodeFields("timeout", timeout))
		}
	}
	options, err := rpchttp.DecodeOptionsFromHeader(header)
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
	next.Header = header
	rpchttp.EncodeOptionsToHeader(next.Header, &rpchttp.Options{Timeout: timeout})
	return next, cancel, nil
}
