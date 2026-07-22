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
	http.Error(o.Response, message, ex.HTTPStatusCode(code))
}
