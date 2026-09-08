package http

import (
	"fmt"
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

// Wire constants and validators are owned by the standalone transport.
const (
	RequestMethod         = rpchttp.RequestMethod
	HeaderAccept          = rpchttp.HeaderAccept
	HeaderAcceptEncoding  = rpchttp.HeaderAcceptEncoding
	HeaderContentType     = rpchttp.HeaderContentType
	HeaderContentEncoding = rpchttp.HeaderContentEncoding
	HeaderContentLength   = rpchttp.HeaderContentLength
	HeaderRpcTrace        = rpchttp.HeaderRpcTrace
	HeaderRpcClient       = rpchttp.HeaderRpcClient
	HeaderRpcActor        = rpchttp.HeaderRpcActor
	HeaderRpcInitiator    = rpchttp.HeaderRpcInitiator
	HeaderRpcOptions      = rpchttp.HeaderRpcOptions
	HeaderRpcStatus       = rpchttp.HeaderRpcStatus
	HeaderRpcServer       = rpchttp.HeaderRpcServer
	ContentTypeJson       = rpchttp.ContentTypeJson
	ContentTypeCbor       = rpchttp.ContentTypeCbor
)

const (
	// MaxRequestBodyBytes is the maximum encoded Rpc request body size.
	MaxRequestBodyBytes int64 = rpchttp.MaxRequestBodyBytes
	// MaxResponseBodyBytes is the maximum encoded Rpc response body size.
	MaxResponseBodyBytes int64 = rpchttp.MaxResponseBodyBytes
)

// ReadRequestBody reads the Rpc request body up to the fixed transport limit.
var ReadRequestBody = rpchttp.ReadRequestBody

// ReadResponseBody reads the Rpc response body up to the fixed transport limit.
var ReadResponseBody = rpchttp.ReadResponseBody

type Options = rpchttp.Options

var CheckRequestMethod = rpchttp.CheckRequestMethod
var MediaTypeOf = rpchttp.MediaTypeOf
var AcceptsContentType = rpchttp.AcceptsContentType
var CheckRequestContentTypeHeader = rpchttp.CheckRequestContentTypeHeader
var CheckRequestHeaders = rpchttp.CheckRequestHeaders
var CheckResponseHeaders = rpchttp.CheckResponseHeaders

func headerValue(header http.Header, key string) (string, bool) {
	values := header.Values(key)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func requestAcceptContentType(methodInfo spec.MethodInfo) string {
	if methodInfo.ResultContainsBinaryType() {
		return ContentTypeCbor + ", " + ContentTypeJson
	}
	return ContentTypeJson
}

func requestBodyContentType(methodInfo spec.MethodInfo) string {
	if methodInfo.ArgumentsContainsBinaryType() {
		return ContentTypeCbor
	}
	return ContentTypeJson
}

func responseBodyContentType(req *http.Request, methodInfo spec.MethodInfo) string {
	if methodInfo.ResultContainsBinaryType() && AcceptsContentType(req.Header.Get(HeaderAccept), ContentTypeCbor) {
		return ContentTypeCbor
	}
	return ContentTypeJson
}

func EncodeContentTypeHeadersToHeaderByMethod(header http.Header, methodInfo spec.MethodInfo) {
	header.Set(HeaderAccept, requestAcceptContentType(methodInfo))
	header.Set(HeaderContentType, requestBodyContentType(methodInfo))
}

func EncodeContentTypeHeadersToHeader(header http.Header, contentType string) {
	header.Set(HeaderContentType, contentType)
}

func DecodeOptionsFromHeader(header http.Header) (*Options, error) {
	return rpchttp.DecodeOptionsFromHeader(header)
}

func EncodeOptionsToHeader(header http.Header, options *Options) {
	rpchttp.EncodeOptionsToHeader(header, options)
}

// Trace Headers

func DecodeTraceFromHeader(header http.Header) (meta.Trace, error) {
	value, ok := headerValue(header, HeaderRpcTrace)
	if !ok {
		return nil, fmt.Errorf("missing request header %s", HeaderRpcTrace)
	}

	trace, err := meta.DecodeTraceFromDelimited(value)
	if err != nil {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcTrace)
	}

	return trace, nil
}

func EncodeTraceToHeader(header http.Header, trace meta.Trace) {
	header.Set(HeaderRpcTrace, rpchttp.EncodeTrace(trace.Id(), trace.Span()))
}

func decodeAppFromHeader(header http.Header, key string) (meta.App, error) {
	value, ok := headerValue(header, key)
	if !ok {
		return nil, fmt.Errorf("missing request header %s", key)
	}

	appInfo, err := meta.DecodeAppFromDelimited(value)
	if err != nil {
		return nil, fmt.Errorf("invalid request header %s", key)
	}
	return appInfo, nil
}

func DecodeClientFromHeader(header http.Header) (meta.App, error) {
	return decodeAppFromHeader(header, HeaderRpcClient)
}

func DecodeServerFromHeader(header http.Header) (meta.App, error) {
	return decodeAppFromHeader(header, HeaderRpcServer)
}

func encodeAppToHeader(header http.Header, appInfo meta.App, key string) {
	header.Set(key, rpchttp.EncodeApp(appInfo.Name(), appInfo.Version(), appInfo.InstanceId()))
}

func EncodeClientToHeader(header http.Header, client meta.App) {
	encodeAppToHeader(header, client, HeaderRpcClient)
}

func EncodeServerToHeader(header http.Header, server meta.App) {
	encodeAppToHeader(header, server, HeaderRpcServer)
}

func DecodeActorFromHeader(header http.Header) (meta.Actor, error) {
	if value, ok := headerValue(header, HeaderRpcActor); ok {
		return meta.DecodeActorFromBase64(value)
	}
	return meta.NewAbsentActor(), nil
}

func EncodeActorToHeader(header http.Header, metaActor meta.Actor) {
	header.Set(HeaderRpcActor, meta.EncodeActorToBase64(metaActor))
}

func DecodeInitiatorFromHeader(header http.Header) (meta.Initiator, error) {
	return meta.DecodeInitiatorFromBase64(header.Get(HeaderRpcInitiator))
}

func EncodeInitiatorToHeader(header http.Header, metaInitiator meta.Initiator) {
	header.Set(HeaderRpcInitiator, meta.EncodeInitiatorToBase64(metaInitiator))
}

// Status Header

var ResponseStatusCode = http.StatusOK

func DecodeStatusCodeFromHeader(header http.Header) (ex.Code, error) {
	value, ok := headerValue(header, HeaderRpcStatus)
	if !ok {
		return "", fmt.Errorf("missing response header %s", HeaderRpcStatus)
	}

	code, err := ex.ParseCode(value)
	if err != nil {
		return "", fmt.Errorf("invalid response header %s", HeaderRpcStatus)
	}
	return code, nil
}

func EncodeStatusCodeToHeader(header http.Header, code ex.Code) {
	vpre.Must(code.IsValid())
	vpre.Must(!code.IsUnresponsive())
	header.Set(HeaderRpcStatus, string(code))
}

// ParseServiceAndMethodFromPath parses only the RPC path segment. Host-level prefixes such
// as "/rpc/invoke" should be removed by the caller before the request reaches http transport.
var ParseServiceAndMethodFromPath = rpchttp.ParseServiceAndMethodFromPath
