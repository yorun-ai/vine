package rpcgw

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

type _HTTPStatusMappingResponseWriter struct {
	http.ResponseWriter
}

func newHTTPStatusMappingResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	return &_HTTPStatusMappingResponseWriter{ResponseWriter: w}
}

func (w *_HTTPStatusMappingResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(mappedHTTPStatusCode(w.Header(), statusCode))
}

func mappedHTTPStatusCode(header http.Header, statusCode int) int {
	code, err := rpchttp.DecodeStatusCodeFromHeader(header)
	if err != nil {
		return statusCode
	}
	return ex.HTTPStatusCode(code)
}
