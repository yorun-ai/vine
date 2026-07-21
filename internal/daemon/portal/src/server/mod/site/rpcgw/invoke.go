package rpcgw

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/computil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	"go.yorun.ai/vine/internal/util/httputil"
)

func (g *RpcGateway) serveInvoke(ctx *spec.Context) {
	invokeRequest := httputil.TrimPathPrefix(ctx.Request, pathInvoke)

	if err := rpchttp.CheckRequestContentTypeHeader(invokeRequest.Header); err != nil {
		logger.Warn("vine.portal rpcgw request content type is invalid", "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.InvalidRequest, "invalid rpc request content type: "+err.Error())
		return
	}

	var cancel context.CancelFunc
	invokeRequest, cancel, err := requestWithRpcOptionsTimeout(invokeRequest, g.context)
	if err != nil {
		logger.Warn("vine.portal rpcgw options is invalid", "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.InvalidRequest, err.Error())
		return
	}
	defer cancel()

	trace, err := ensureRpcTrace(invokeRequest)
	if err != nil {
		logger.Warn("vine.portal rpcgw trace is invalid", "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.InvalidRequest, err.Error())
		return
	}

	initiator, err := ensureRpcInitiator(invokeRequest, ctx.RemoteAddr)
	if err != nil {
		logger.Warn("vine.portal rpcgw initiator cannot be created", "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.InvalidRequest, "rpc request initiator cannot be created: "+err.Error())
		return
	}

	serviceName, methodName, err := rpchttp.ParseServiceAndMethodFromPath(invokeRequest.URL.Path)
	if err != nil {
		logger.Warn("vine.portal rpcgw invoke path is invalid", "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.InvalidRequest, "invalid rpc request path")
		return
	}

	operation := &access.RpcOperation{
		Auther: access.Auther{
			Request:   invokeRequest,
			Response:  ctx.ResponseWriter,
			Trace:     trace,
			Initiator: initiator,
		},
		Server:      g.app,
		ActorVia:    g.actorVia,
		ServiceName: serviceName,
		MethodName:  methodName,
	}
	if !g.access.AllowRpc(operation) {
		return
	}

	registration, configured := g.routeService(serviceName)
	if !configured {
		logger.Warn("vine.portal rpcgw service is not configured", "site", g.name, "service", serviceName)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.NotFound, "rpcgw service is not configured: "+serviceName)
		return
	}
	if registration == nil {
		logger.Warn("vine.portal rpcgw service endpoint is unavailable", "site", g.name, "service", serviceName)
		g.writeError(ctx.ResponseWriter, invokeRequest, ex.ServiceUnavailable, "rpcgw service endpoint is unavailable: "+serviceName)
		return
	}

	g.forwardInvoke(&spec.Context{
		Request:        invokeRequest,
		ResponseWriter: ctx.ResponseWriter,
		RemoteAddr:     ctx.RemoteAddr,
	}, registration.Endpoint)
}

func (g *RpcGateway) forwardInvoke(ctx *spec.Context, endpoint string) {
	acceptEncoding, forwardRequest := prepareForwardRequest(ctx.Request)
	response, err := gwutil.ForwardRequest(forwardRequest, endpoint)
	if err != nil {
		statusCode := ex.ServiceUnavailable
		if errors.Is(err, context.DeadlineExceeded) {
			statusCode = ex.GatewayTimeout
		} else if errors.Is(err, gwutil.ErrInvalidForwardRequest) {
			statusCode = ex.InvalidRequest
		}
		logger.Warn("vine.portal rpcgw forward failed", "endpoint", endpoint, "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, ctx.Request, statusCode, "rpcgw forward failed: "+err.Error())
		return
	}

	defer func() { _ = response.Body.Close() }()
	body, err := httputil.ReadResponseBody(response)
	if err != nil {
		logger.Warn("vine.portal rpcgw response body cannot be read", "endpoint", endpoint, "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, ctx.Request, ex.ServiceUnavailable, "rpcgw response body cannot be read: "+err.Error())
		return
	}
	body, err = rpchttp.ClearResponseErrorDetail(body, response.Header.Get(rpchttp.HeaderContentType))
	if err != nil {
		logger.Warn("vine.portal rpcgw response body cannot be parsed", "endpoint", endpoint, "path", ctx.Request.URL.Path, "error", err)
		g.writeError(ctx.ResponseWriter, ctx.Request, ex.UnexpectedResponse, "rpcgw response body cannot be parsed")
		return
	}
	httputil.CopyHeader(ctx.ResponseWriter.Header(), response.Header)
	httputil.ClearBodyHeaders(ctx.ResponseWriter.Header())
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, rpcTraceIdOrNew(ctx.Request.Header))
	body, contentEncoding := computil.CompressResponseBody(body, acceptEncoding)
	if contentEncoding != "" {
		ctx.ResponseWriter.Header().Set(rpchttp.HeaderContentEncoding, contentEncoding)
		ctx.ResponseWriter.Header().Add("Vary", rpchttp.HeaderAcceptEncoding)
	}
	ctx.ResponseWriter.WriteHeader(response.StatusCode)
	_, _ = ctx.ResponseWriter.Write(body)
}

func ensureRpcTrace(request *http.Request) (meta.Trace, error) {
	traceValue := request.Header.Get(rpchttp.HeaderRpcTrace)
	if traceValue == "" {
		return nil, fmt.Errorf("missing request header %s", rpchttp.HeaderRpcTrace)
	}
	trace, err := meta.DecodeTraceFromDelimitedOrNewSpan(traceValue)
	if err != nil {
		return nil, fmt.Errorf("invalid request header %s", rpchttp.HeaderRpcTrace)
	}
	gatewayTrace := trace.NewChildTrace()
	rpchttp.EncodeTraceToHeader(request.Header, gatewayTrace)
	return gatewayTrace, nil
}
func ensureRpcInitiator(request *http.Request, remoteAddr string) (meta.Initiator, error) {
	if request.Header.Get(rpchttp.HeaderRpcClient) == "" {
		return nil, fmt.Errorf("missing request header %s", rpchttp.HeaderRpcClient)
	}

	client, err := rpchttp.DecodeClientFromHeader(request.Header)
	if err != nil {
		return nil, err
	}
	initiator, err := meta.NewInitiator(
		client.Name(),
		client.Version(),
		client.InstanceId(),
		request.UserAgent(),
		remoteAddr,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set(rpchttp.HeaderRpcInitiator, meta.EncodeInitiatorToBase64(initiator))
	return initiator, nil
}
