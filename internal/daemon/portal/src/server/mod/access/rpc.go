package access

import (
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vpre"
)

const defaultAuthMode = skel.AuthModeRequired

type RpcOperation struct {
	Auther

	Server meta.App

	ActorVia    watched.PortalActorVia
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

	mode := o.authMode()
	if mode == skel.AuthModeOff {
		o.writeError(ex.ServiceUnavailable, "Rpc does not support auth off")
		return false
	}
	if !o.authenticate(mode, o.writeError, o.setActor) {
		return false
	}
	return true
}

func (o *RpcOperation) setActor(actor meta.Actor) {
	o.actor = actor
	o.Request.Header.Set(rpchttp.HeaderRpcActor, meta.EncodeActorToBase64(actor))
}

func (o *RpcOperation) writeError(code ex.Code, message string, options ...ex.ErrorOption) {
	vpre.MustNil(rpchttp.WriteRequestErrorResponse(o.Response, o.Request, o.Server, ex.New(code, message, options...)))
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
	if authMode == "" || authMode == skel.AuthModeUnset {
		authMode = o.serviceSchema.AuthMode
	}
	if authMode == "" || authMode == skel.AuthModeUnset {
		authMode = defaultAuthMode
	}
	return authMode
}

func (o *RpcOperation) writeErrorWithReason(code ex.Code, message string, reason string) {
	var options []ex.ErrorOption
	if reason != "" {
		options = append(options, ex.WithReason(reason))
	}
	o.writeError(code, message, options...)
}
