package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

type InvokeRequest struct {
	Context         context.Context
	Endpoint        string
	ServiceSkelName string
	MethodSkelName  string
	Params          any
	Trace           meta.Trace
	Client          meta.App
	Actor           meta.Actor
	Initiator       meta.Initiator
}

func BuildInvokeRequest(invokeRequest InvokeRequest) *http.Request {
	body := vcode.MustMarshalJson(map[string]any{
		"params": invokeRequest.Params,
	})

	request, err := http.NewRequestWithContext(
		invokeRequest.Context,
		RequestMethod,
		formatInvokeURL(invokeRequest.Endpoint, invokeRequest.ServiceSkelName, invokeRequest.MethodSkelName),
		bytes.NewReader(body),
	)
	vpre.MustNil(err)

	header := request.Header
	EncodeJsonContentTypeHeadersToHeader(header)
	EncodeRequestOptionsToHeader(header, invokeRequest.Context)
	EncodeTraceToHeader(header, invokeRequest.Trace)
	EncodeClientToHeader(header, invokeRequest.Client)
	if invokeRequest.Actor != nil {
		EncodeActorToHeader(header, invokeRequest.Actor)
	}
	if invokeRequest.Initiator != nil {
		EncodeInitiatorToHeader(header, invokeRequest.Initiator)
	}
	return request
}

func formatInvokeURL(endpoint string, serviceSkelName string, methodSkelName string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), serviceSkelName, methodSkelName)
}
