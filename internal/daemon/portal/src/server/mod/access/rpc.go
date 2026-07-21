package access

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vpre"
)

const defaultAuthMode = skel.AuthModeAuth

type RpcOperation struct {
	Auther

	Server meta.App

	ActorVia    redised.PortalActorVia
	ServiceName string
	MethodName  string

	serviceSchema *skel.ServiceSchema
	methodSchema  *skel.MethodSchema
	requestBody   []byte
	cborPayload   any

	permissionCodeResults map[string]bool
}

func (o *RpcOperation) Auth() bool {
	if !o.loadMethodSchema() {
		return false
	}

	if o.authMode() == skel.AuthModeNoAuth {
		o.actor = meta.NewAnonymousActor()
		o.Request.Header.Set(rpchttp.HeaderRpcActor, meta.EncodeActorToBase64(o.actor))
		return true
	}

	if !o.actorSchema.AuthEnabled {
		o.writeError(ex.ClientForbidden, "rpc method require auth, but actor auth not enabled")
		return false
	}

	return o.auth(o.writeError, o.setActor)
}

func (o *RpcOperation) setActor(actor meta.Actor) {
	o.actor = actor
	o.Request.Header.Set(rpchttp.HeaderRpcActor, meta.EncodeActorToBase64(actor))
}

func (o *RpcOperation) writeError(code ex.Code, message string) {
	vpre.MustNil(rpchttp.WriteRequestErrorResponse(o.Response, o.Request, o.Server, ex.New(code, message)))
}

func (o *RpcOperation) loadMethodSchema() bool {
	if o.methodSchema != nil {
		return true
	}

	method, ok := o.serviceSchema.MethodByName(o.MethodName)
	if !ok {
		o.writeError(ex.NotFound, "rpc method schema is not found: "+o.ServiceName+"/"+o.MethodName)
		return false
	}

	o.methodSchema = method
	return true
}

func (o *RpcOperation) authMode() skel.AuthMode {
	authMode := o.methodSchema.AuthMode
	if authMode == skel.AuthModeUnset {
		authMode = o.serviceSchema.AuthMode
	}
	if authMode == skel.AuthModeUnset {
		authMode = defaultAuthMode
	}
	return authMode
}
