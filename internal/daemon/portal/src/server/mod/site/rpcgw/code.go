package rpcgw

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

func rpcHTTPStatusCode(code ex.Code) int {
	switch code {
	case ex.OK:
		return http.StatusOK
	case ex.InvalidRequest:
		return http.StatusBadRequest
	case ex.Unauthorized:
		return http.StatusUnauthorized
	case ex.ClientForbidden, ex.PermissionDenied, ex.ElevationRequired:
		return http.StatusForbidden
	case ex.NotFound:
		return http.StatusNotFound
	case ex.ValidationFailed, ex.OperationFailed:
		return http.StatusUnprocessableEntity
	case ex.ServiceUnavailable:
		return http.StatusServiceUnavailable
	case ex.GatewayTimeout:
		return http.StatusGatewayTimeout
	case ex.Internal, ex.Unknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

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
	return rpcHTTPStatusCode(code)
}
