package http

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/rpc/spec"
)

func setTestRequestContentTypeHeaders(header http.Header, methodInfo spec.MethodInfo) {
	header.Set(HeaderAccept, requestAcceptContentType(methodInfo))
	header.Set(HeaderContentType, requestBodyContentType(methodInfo))
}
