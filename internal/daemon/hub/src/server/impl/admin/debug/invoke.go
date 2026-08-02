package debug

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/util/httputil"
)

var serviceDebugHTTPClient = httputil.NewH2CClient()

func doServiceDebugInvokeRequest(request *http.Request) (*http.Response, error) {
	ctx, cancel := httputil.ContextWithForwardTimeout(request)
	defer cancel()

	forwardRequest := request.Clone(ctx)
	forwardRequest.Header = request.Header.Clone()

	var response *http.Response
	var err error
	if ingressinproc.IsEndpoint(request.URL.String()) {
		response, err = ingressinproc.RoundTrip(request.URL.String(), forwardRequest)
	} else {
		response, err = serviceDebugHTTPClient.Do(forwardRequest)
	}
	if err != nil {
		return nil, normalizeServiceDebugForwardError(ctx, err)
	}
	return response, nil
}

func normalizeServiceDebugForwardError(ctx context.Context, err error) error {
	if ctx.Err() == context.DeadlineExceeded {
		return context.DeadlineExceeded
	}
	return err
}

func debugTrace(traceId *string, spanId *string) meta.Trace {
	if traceId == nil || strings.TrimSpace(*traceId) == "" {
		return meta.InitialTrace()
	}
	span := ""
	if spanId != nil {
		span = strings.TrimSpace(*spanId)
	}
	trace, err := meta.NewTrace(strings.TrimSpace(*traceId), span)
	ex.PanicNewIfError(err, ex.InvalidRequest)
	return trace
}

func (s *ServiceDebugServiceServerImpl) debugActor(actorSkelName *string, actorInfoJson skel.JSON) meta.Actor {
	if actorSkelName == nil || strings.TrimSpace(*actorSkelName) == "" {
		return nil
	}
	actor := s.findActorSchema(strings.TrimSpace(*actorSkelName))
	ex.PanicNewIfNot(actor.AuthInfo != nil, ex.InvalidRequest, "actor auth info schema not found")
	info := json.RawMessage(strings.TrimSpace(string(actorInfoJson)))
	if len(info) == 0 {
		info = json.RawMessage("{}")
	}
	if !json.Valid(info) {
		ex.PanicNew(ex.InvalidRequest, "invalid actor info json")
	}
	return meta.NewAuthenticatedActorWithRawInfo(actor.AuthInfo.SkelName, info)
}

const (
	debugClientName       = "vine.hub.debug"
	debugClientVersion    = "0.0.0"
	debugClientInstanceId = "00000000-0000-0000-0000-000000000000"
)

func debugClient() meta.App {
	return meta.MustNewApp(debugClientName, debugClientVersion, debugClientInstanceId)
}
