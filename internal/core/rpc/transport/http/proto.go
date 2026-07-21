package http

import (
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vstring"
)

// HTTP method
const RequestMethod = http.MethodPost

func CheckRequestMethod(req *http.Request) error {
	if req.Method != RequestMethod {
		return fmt.Errorf("invalid request method, expected %s", RequestMethod)
	}
	return nil
}

// Header definitions
const (
	HeaderAccept         = "accept"
	HeaderAcceptEncoding = "accept-encoding"

	HeaderContentType     = "content-type"
	HeaderContentEncoding = "content-encoding"
	HeaderContentLength   = "content-length"

	HeaderRpcTrace = "vrpc-trace"

	HeaderRpcClient    = "vrpc-client"
	HeaderRpcActor     = "vrpc-actor"
	HeaderRpcInitiator = "vrpc-initiator"
	HeaderRpcOptions   = "vrpc-options"

	HeaderRpcStatus = "vrpc-status"
	HeaderRpcServer = "vrpc-server"
)

// Content types
const (
	ContentTypeJson = "application/vrpc+json"
	ContentTypeCbor = "application/vrpc+cbor"
)

var supportedContentTypes = []string{ContentTypeJson, ContentTypeCbor}

func MediaTypeOf(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err == nil {
		return mediaType
	}
	mediaType, _, _ = strings.Cut(value, ";")
	return strings.TrimSpace(mediaType)
}

func IsValidContentType(value string) bool {
	value = MediaTypeOf(value)
	for _, contentType := range supportedContentTypes {
		if value == contentType {
			return true
		}
	}
	return false
}

func AcceptsValidContentType(value string) bool {
	for _, accepted := range strings.Split(value, ",") {
		if IsValidContentType(accepted) {
			return true
		}
	}
	return false
}

func AcceptsContentType(value string, contentType string) bool {
	for _, accepted := range strings.Split(value, ",") {
		if MediaTypeOf(accepted) == contentType {
			return true
		}
	}
	return false
}

// Header checks and fixed headers
func headerValue(header http.Header, key string) (value string, ok bool) {
	if values := header.Values(key); len(values) > 0 {
		return values[0], true
	}
	return "", false
}

var (
	requiredRequestHeaders = []string{
		HeaderAccept,
		HeaderContentType,
		HeaderRpcTrace,
		HeaderRpcClient,
	}
	requiredResponseHeaders = []string{
		HeaderContentType,
		HeaderRpcStatus,
		HeaderRpcServer,
	}
)

func newMissingHeaderError(scope string, key string) error {
	return fmt.Errorf("missing %s header %s", scope, key)
}

func newDuplicatedHeaderError(scope string, key string) error {
	return fmt.Errorf("invalid %s header %s: duplicated", scope, key)
}

func newInvalidHeaderError(scope string, key string, value string) error {
	return fmt.Errorf("invalid %s header %s: %s", scope, key, value)
}

func checkDuplicatedHeaders(header http.Header, scope string) error {
	for key, values := range header {
		if len(values) > 1 {
			return newDuplicatedHeaderError(scope, key)
		}
	}
	return nil
}

func checkRequiredHeaders(header http.Header, scope string, required []string) error {
	for _, key := range required {
		_, ok := headerValue(header, key)
		if !ok {
			return newMissingHeaderError(scope, key)
		}
	}
	return nil
}

func checkContentTypeHeader(header http.Header, scope string) error {
	value, ok := headerValue(header, HeaderContentType)
	if !ok {
		return newMissingHeaderError(scope, HeaderContentType)
	}
	if !IsValidContentType(value) {
		return newInvalidHeaderError(scope, HeaderContentType, value)
	}
	return nil
}

func CheckRequestContentTypeHeader(header http.Header) error {
	return checkContentTypeHeader(header, "request")
}

func checkAcceptHeader(header http.Header, scope string) error {
	value, ok := headerValue(header, HeaderAccept)
	if !ok {
		return newMissingHeaderError(scope, HeaderAccept)
	}
	if !AcceptsValidContentType(value) {
		return newInvalidHeaderError(scope, HeaderAccept, value)
	}
	return nil
}

func CheckRequestHeaders(header http.Header) error {
	if err := checkDuplicatedHeaders(header, "request"); err != nil {
		return err
	}
	if err := checkRequiredHeaders(header, "request", requiredRequestHeaders); err != nil {
		return err
	}
	if err := checkAcceptHeader(header, "request"); err != nil {
		return err
	}
	return checkContentTypeHeader(header, "request")
}

func CheckResponseHeaders(header http.Header) error {
	if err := checkDuplicatedHeaders(header, "response"); err != nil {
		return err
	}
	if err := checkRequiredHeaders(header, "response", requiredResponseHeaders); err != nil {
		return err
	}
	return checkContentTypeHeader(header, "response")
}

type Options struct {
	Timeout time.Duration
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

func EncodeJsonContentTypeHeadersToHeader(header http.Header) {
	header.Set(HeaderAccept, ContentTypeJson)
	header.Set(HeaderContentType, ContentTypeJson)
}

func EncodeContentTypeHeadersToHeader(header http.Header, contentType string) {
	header.Set(HeaderContentType, contentType)
}

func DecodeOptionsFromHeader(header http.Header) (*Options, error) {
	value, ok := headerValue(header, HeaderRpcOptions)
	if !ok {
		return &Options{}, nil
	}

	fields, err := vstring.DecodeDelimited(value)
	if err != nil {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	timeoutValue := fields["timeout"]
	if timeoutValue == "" || len(fields) != 1 {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil || timeout <= 0 {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	return &Options{Timeout: timeout}, nil
}

func EncodeOptionsToHeader(header http.Header, options *Options) {
	if options == nil || options.Timeout <= 0 {
		return
	}
	header.Set(HeaderRpcOptions, vstring.EncodeDelimited("timeout", options.Timeout.String()))
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
	header.Set(HeaderRpcTrace, meta.EncodeTraceToDelimited(trace))
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
	header.Set(key, meta.EncodeAppToDelimited(appInfo))
}

func logicalAppName(name string) string {
	name, _, _ = strings.Cut(name, "@")
	return name
}

func EncodeClientToHeader(header http.Header, client meta.App) {
	encodeAppToHeader(header, client, HeaderRpcClient)
}

func EncodeServerToHeader(header http.Header, server meta.App) {
	header.Set(HeaderRpcServer, vstring.EncodeDelimited(
		"name", logicalAppName(server.Name()),
		"version", server.Version(),
		"instanceId", server.InstanceId(),
	))
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

// Request path
var requestPathRegexp = regexp.MustCompile(`^/([\w.]+)/(\w+)$`)

// ParseServiceAndMethodFromPath parses only the RPC path segment. Host-level prefixes such
// as "/rpc/invoke" should be removed by the caller before the request reaches http transport.
func ParseServiceAndMethodFromPath(path string) (serviceName string, methodName string, err error) {
	ms := requestPathRegexp.FindStringSubmatch(path)
	if ms == nil {
		return "", "", fmt.Errorf("invalid request path: %s", path)
	}
	return ms[1], ms[2], nil
}
