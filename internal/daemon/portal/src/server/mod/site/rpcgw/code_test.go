package rpcgw

import (
	"net/http"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
)

func TestRpcHTTPStatusCode(t *testing.T) {
	cases := []struct {
		code       ex.Code
		statusCode int
	}{
		{code: ex.OK, statusCode: http.StatusOK},
		{code: ex.InvalidRequest, statusCode: http.StatusBadRequest},
		{code: ex.Unauthorized, statusCode: http.StatusUnauthorized},
		{code: ex.ClientForbidden, statusCode: http.StatusForbidden},
		{code: ex.PermissionDenied, statusCode: http.StatusForbidden},
		{code: ex.ElevationRequired, statusCode: http.StatusForbidden},
		{code: ex.NotFound, statusCode: http.StatusNotFound},
		{code: ex.ValidationFailed, statusCode: http.StatusUnprocessableEntity},
		{code: ex.OperationFailed, statusCode: http.StatusUnprocessableEntity},
		{code: ex.ServiceUnavailable, statusCode: http.StatusServiceUnavailable},
		{code: ex.GatewayTimeout, statusCode: http.StatusGatewayTimeout},
		{code: ex.Internal, statusCode: http.StatusInternalServerError},
		{code: ex.Unknown, statusCode: http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := rpcHTTPStatusCode(tc.code); got != tc.statusCode {
			t.Fatalf("rpcHTTPStatusCode(%s) = %d, want %d", tc.code, got, tc.statusCode)
		}
	}
}
