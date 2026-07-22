package ex

import "net/http"

// HTTPStatusCode maps a structured error code to its canonical HTTP status.
func HTTPStatusCode(code Code) int {
	switch code {
	case OK:
		return http.StatusOK
	case InvalidRequest:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case ClientForbidden, PermissionDenied, ElevationRequired:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case ValidationFailed, OperationFailed:
		return http.StatusUnprocessableEntity
	case ServiceUnavailable:
		return http.StatusServiceUnavailable
	case GatewayTimeout:
		return http.StatusGatewayTimeout
	case Internal, Unknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
