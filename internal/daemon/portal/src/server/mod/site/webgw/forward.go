package webgw

import (
	"context"
	"errors"
	"io"
	"net/http"

	"go.yorun.ai/vine/internal/core/logger"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/computil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	"go.yorun.ai/vine/internal/util/httputil"
)

var webGatewayLogger = logger.New("daemon:portal:webgw")

func (g *WebGateway) forward(ctx *spec.Context, request *http.Request, registration *watched.WebRegistration, traceId string) {
	endpoint := registration.Endpoint
	encodeWebOptionsToHeader(request)
	encodeWebForwardTrace(request.Header)
	transport, err := g.access.Identity.BackendTransport(registration.ServerIdentity.SPIFFEPath(), endpoint)
	if err != nil {
		http.Error(ctx.ResponseWriter, "webgw endpoint is insecure: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	if handled, err := gwutil.ForwardUpgradeWithTransport(ctx.ResponseWriter, request, endpoint, transport, webGatewayLogger); handled {
		if err != nil {
			http.Error(ctx.ResponseWriter, "webgw websocket forward failed: "+err.Error(), http.StatusServiceUnavailable)
		}
		return
	}
	response, err := gwutil.ForwardRequestWithTransport(request, endpoint, transport)
	if err != nil {
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			statusCode = http.StatusBadGateway
		}
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
	// These statuses carry no body; do not advertise a transfer length to HTTP/2.
	if response.StatusCode == http.StatusNotModified || response.StatusCode == http.StatusNoContent {
		httputil.ClearBodyHeaders(ctx.ResponseWriter.Header())
	}
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, traceId)
	ctx.ResponseWriter.WriteHeader(response.StatusCode)
	response.Request = request
	if transferred, err := httputil.CopyResponseBody(ctx.ResponseWriter, response); err != nil {
		expected, known := httputil.ResponseBodyLength(response)
		httputil.LogResponseTransferError(ctx.Request.Context(), webGatewayLogger, err, "phase", "body", "expectedLength", expected, "lengthKnown", known, "transferredBytes", transferred)
		panic(http.ErrAbortHandler)
	}
}

func (g *WebGateway) writeCompressedResponse(ctx *spec.Context, response *http.Response, traceId string) {
	body, err := httputil.ReadResponseBody(response)
	if err != nil {
		expected, known := httputil.ResponseBodyLength(response)
		httputil.LogResponseTransferError(ctx.Request.Context(), webGatewayLogger, err, "phase", "read", "expectedLength", expected, "lengthKnown", known, "transferredBytes", len(body))
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		http.Error(ctx.ResponseWriter, "webgw response body cannot be read", status)
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
	if n, err := ctx.ResponseWriter.Write(body); err != nil || n != len(body) {
		if err == nil {
			err = io.ErrShortWrite
		}
		httputil.LogResponseTransferError(ctx.Request.Context(), webGatewayLogger, err, "phase", "write", "expectedLength", len(body), "transferredBytes", n)
		panic(http.ErrAbortHandler)
	}
}

func encodeWebForwardTrace(header http.Header) {
	trace, err := webspec.DecodeTraceFromHeader(header)
	if err != nil {
		return
	}
	webspec.EncodeTraceToHeader(header, trace.NewChildTrace())
}
