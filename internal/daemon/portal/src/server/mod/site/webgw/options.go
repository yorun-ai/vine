package webgw

import (
	"context"
	"fmt"
	"net/http"
	"time"

	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	"go.yorun.ai/vine/internal/util/httputil"
)

const (
	defaultWebTimeout = 30 * time.Second
	maxWebTimeout     = 120 * time.Second
)

func requestWithWebOptionsTimeout(request *http.Request, gatewayContext context.Context) (*http.Request, context.CancelFunc, error) {
	options, err := webspec.DecodeOptionsFromHeader(request.Header)
	if err != nil {
		return request, nil, err
	}

	timeout := options.Timeout
	if timeout <= 0 && (httputil.IsUpgradeRequest(request) || httputil.IsEventStreamRequest(request)) {
		ctx, cancel := context.WithCancel(request.Context())
		stopGatewayCancel := context.AfterFunc(gatewayContext, cancel)
		next := request.Clone(ctx)
		next.Header = request.Header.Clone()
		return next, func() {
			stopGatewayCancel()
			cancel()
		}, nil
	}
	if timeout <= 0 {
		timeout = defaultWebTimeout
	}
	if timeout > maxWebTimeout {
		return request, nil, fmt.Errorf("invalid header %s: timeout exceeds %s", webspec.HeaderWebOptions, maxWebTimeout)
	}

	ctx, cancel := gwutil.ContextWithoutClientCancel(request.Context(), gatewayContext, timeout)
	next := request.Clone(ctx)
	next.Header = request.Header.Clone()
	webspec.EncodeOptionsToHeader(next.Header, &webspec.Options{Timeout: timeout})
	return next, cancel, nil
}

func encodeWebOptionsToHeader(request *http.Request) {
	request.Header.Del(webspec.HeaderWebOptions)
	webspec.EncodeRequestOptionsToHeader(request.Header, request.Context())
}
