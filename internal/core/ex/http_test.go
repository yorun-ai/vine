package ex

import (
	"net/http"
	"testing"
)

func TestHTTPStatusCode(t *testing.T) {
	testCases := []struct {
		code   Code
		status int
	}{
		{code: OK, status: http.StatusOK},
		{code: InvalidRequest, status: http.StatusBadRequest},
		{code: Unauthorized, status: http.StatusUnauthorized},
		{code: ClientForbidden, status: http.StatusForbidden},
		{code: PermissionDenied, status: http.StatusForbidden},
		{code: ElevationRequired, status: http.StatusForbidden},
		{code: NotFound, status: http.StatusNotFound},
		{code: ValidationFailed, status: http.StatusUnprocessableEntity},
		{code: OperationFailed, status: http.StatusUnprocessableEntity},
		{code: ServiceUnavailable, status: http.StatusServiceUnavailable},
		{code: GatewayTimeout, status: http.StatusGatewayTimeout},
		{code: Internal, status: http.StatusInternalServerError},
		{code: Unknown, status: http.StatusInternalServerError},
		{code: Code("INVALID"), status: http.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		if got := HTTPStatusCode(testCase.code); got != testCase.status {
			t.Fatalf("HTTPStatusCode(%s) = %d, want %d", testCase.code, got, testCase.status)
		}
	}
}
