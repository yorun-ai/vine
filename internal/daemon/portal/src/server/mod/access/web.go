package access

import (
	"net/http"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

type WebOperation struct {
	Auther

	ActorVia watched.PortalActorVia
	WebName  string
}

func (o *WebOperation) Auth() bool {
	if !o.auth(o.writeError, o.setActor) {
		return false
	}
	o.Request.Header.Del(headerAuthorization)
	return true
}

func (o *WebOperation) setActor(actor meta.Actor) {
	o.actor = actor
	o.Request.Header.Set(webspec.HeaderWebActor, meta.EncodeActorToBase64(actor))
}

func (o *WebOperation) writeError(code ex.Code, message string, options ...ex.ErrorOption) {
	http.Error(o.Response, message, ex.HTTPStatusCode(code))
}
