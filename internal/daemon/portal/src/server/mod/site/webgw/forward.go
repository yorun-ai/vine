package webgw

import (
	"context"
	"errors"
	"net/http"

	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/computil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	"go.yorun.ai/vine/internal/util/httputil"
)

func (g *WebGateway) forward(ctx *spec.Context, request *http.Request, endpoint string, traceId string) {
	encodeWebOptionsToHeader(request)
	encodeWebForwardTrace(request.Header)
	if handled, err := gwutil.ForwardUpgrade(ctx.ResponseWriter, request, endpoint); handled {
		if err != nil {
			http.Error(ctx.ResponseWriter, "webgw websocket forward failed: "+err.Error(), http.StatusServiceUnavailable)
		}
		return
	}
	response, err := gwutil.ForwardRequest(request, endpoint)
	if err != nil {
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, context.DeadlineExceeded) {
			statusCode = http.StatusGatewayTimeout
		}
		http.Error(ctx.ResponseWriter, "webgw forward failed: "+err.Error(), statusCode)
		return
	}

	defer func() { _ = response.Body.Close() }()
	if computil.ShouldCompressWebResponse(ctx.Request, response) {
		g.writeCompressedResponse(ctx, response, traceId)
		return
	}

	httputil.CopyHeader(ctx.ResponseWriter.Header(), response.Header)
	httputil.ClearBodyHeaders(ctx.ResponseWriter.Header())
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, traceId)
	ctx.ResponseWriter.WriteHeader(response.StatusCode)
	httputil.CopyResponseBody(ctx.ResponseWriter, response)
}

func (g *WebGateway) writeCompressedResponse(ctx *spec.Context, response *http.Response, traceId string) {
	body, err := httputil.ReadResponseBody(response)
	if err != nil {
		http.Error(ctx.ResponseWriter, "webgw response body cannot be read: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	httputil.CopyHeader(ctx.ResponseWriter.Header(), response.Header)
	httputil.ClearBodyHeaders(ctx.ResponseWriter.Header())
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, traceId)
	body, contentEncoding := computil.CompressResponseBody(body, ctx.Request.Header.Get("Accept-Encoding"))
	if contentEncoding != "" {
		ctx.ResponseWriter.Header().Set("Content-Encoding", contentEncoding)
		ctx.ResponseWriter.Header().Add("Vary", "Accept-Encoding")
	}
	ctx.ResponseWriter.WriteHeader(response.StatusCode)
	_, _ = ctx.ResponseWriter.Write(body)
}

func encodeWebForwardTrace(header http.Header) {
	trace, err := webspec.DecodeTraceFromHeader(header)
	if err != nil {
		return
	}
	webspec.EncodeTraceToHeader(header, trace.NewChildTrace())
}
