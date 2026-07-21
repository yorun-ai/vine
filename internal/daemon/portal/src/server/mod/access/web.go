package access

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

type WebOperation struct {
	Auther

	ActorVia redised.PortalActorVia
}

func (o *WebOperation) Auth() bool {
	return o.auth(o.writeError, o.setActor)
}

func (o *WebOperation) setActor(actor meta.Actor) {
	o.actor = actor
	o.Request.Header.Set(webspec.HeaderWebActor, meta.EncodeActorToBase64(actor))
}

func (o *WebOperation) writeError(code ex.Code, message string) {
	http.Error(o.Response, message, webHTTPStatusCode(code))
}

func webHTTPStatusCode(code ex.Code) int {
	switch code {
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
	case ex.GatewayTimeout:
		return http.StatusGatewayTimeout
	case ex.ServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
