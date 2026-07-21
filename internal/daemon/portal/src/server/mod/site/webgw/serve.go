package webgw

import (
	"context"
	"net/http"

	"go.yorun.ai/vine/internal/core/meta"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

const (
	defaultWebInitiatorName       = "vine.portal"
	defaultWebInitiatorVersion    = "0.0.0"
	defaultWebInitiatorInstanceId = "00000000-0000-0000-0000-000000000001"
)

var optionsAllowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

func (g *WebGateway) Serve(ctx *spec.Context) {
	if ctx.Request.Method == http.MethodOptions {
		spec.ServeOptions(ctx.ResponseWriter, ctx.Request, g.cors, ctx.EntryOrigin, optionsAllowedMethods)
		return
	}
	if spec.ApplyCORS(ctx.ResponseWriter, ctx.Request, g.cors, ctx.EntryOrigin) {
		spec.ExposeHeader(ctx.ResponseWriter.Header(), spec.HeaderPortalTraceId)
	}

	request := ctx.Request.Clone(ctx.Request.Context())
	request.Header = ctx.Request.Header.Clone()
	var cancel context.CancelFunc
	request, cancel, err := requestWithWebOptionsTimeout(request, g.context)
	if err != nil {
		ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, webTraceIdOrNew(request.Header))
		http.Error(ctx.ResponseWriter, "web request options cannot be created: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer cancel()

	trace, err := ensureWebTrace(request)
	if err != nil {
		ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, meta.NewId())
		http.Error(ctx.ResponseWriter, "web request meta cannot be created: "+err.Error(), http.StatusBadRequest)
		return
	}
	ctx.ResponseWriter.Header().Set(spec.HeaderPortalTraceId, trace.Id())

	registration, configured := g.routeWeb()
	if !configured {
		http.Error(ctx.ResponseWriter, "webgw is not configured: "+g.name, http.StatusNotFound)
		return
	}

	if registration == nil {
		http.Error(ctx.ResponseWriter, "webgw endpoint is unavailable: "+g.name, http.StatusServiceUnavailable)
		return
	}

	initiator, err := ensureWebInitiator(request, ctx.RemoteAddr)
	if err != nil {
		http.Error(ctx.ResponseWriter, "web request meta cannot be created: "+err.Error(), http.StatusBadRequest)
		return
	}

	operation := &access.WebOperation{
		Auther: access.Auther{
			Request:   request,
			Response:  ctx.ResponseWriter,
			Trace:     trace,
			Initiator: initiator,
		},
		ActorVia: g.actorVia,
	}
	if !g.access.AuthWeb(operation) {
		return
	}

	g.forward(ctx, request, registration.Endpoint, trace.Id())
}

func ensureWebTrace(request *http.Request) (meta.Trace, error) {
	traceValue := request.Header.Get(webspec.HeaderWebTrace)
	if traceValue == "" {
		trace := meta.InitialTrace()
		gatewayTrace := trace.NewChildTrace()
		webspec.EncodeTraceToHeader(request.Header, gatewayTrace)
		return gatewayTrace, nil
	}

	trace, err := meta.DecodeTraceFromDelimitedOrNewSpan(traceValue)
	if err != nil {
		return nil, err
	}
	gatewayTrace := trace.NewChildTrace()
	webspec.EncodeTraceToHeader(request.Header, gatewayTrace)
	return gatewayTrace, nil
}

func webTraceIdOrNew(header http.Header) string {
	traceValue := header.Get(webspec.HeaderWebTrace)
	if traceValue == "" {
		return meta.NewId()
	}
	trace, err := meta.DecodeTraceFromDelimitedOrNewSpan(traceValue)
	if err != nil {
		return meta.NewId()
	}
	return trace.Id()
}

func ensureWebInitiator(request *http.Request, remoteAddr string) (meta.Initiator, error) {
	initiator, err := meta.NewInitiator(
		defaultWebInitiatorName,
		defaultWebInitiatorVersion,
		defaultWebInitiatorInstanceId,
		request.UserAgent(),
		remoteAddr,
	)
	if err != nil {
		return nil, err
	}

	request.Header.Set(webspec.HeaderWebInitiator, meta.EncodeInitiatorToBase64(initiator))
	return initiator, nil
}
