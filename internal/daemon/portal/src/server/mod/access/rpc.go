package access

import (
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vpre"
)

const defaultAuthMode = skeldesc.AuthModeRequired

type RpcOperation struct {
	Auther

	Server meta.App

	ActorVia          watched.PortalActorVia
	ServiceName       string
	MethodName        string
	serviceDescriptor *skeldesc.Service
	methodDescriptor  *skeldesc.Method
	requestBody       []byte
	cborPayload       any

	permissionCodeResults map[string]bool
}

func (o *RpcOperation) Auth() bool {
	if !o.loadMethodDescriptor() {
		return false
	}

	mode := o.methodDescriptor.EffectiveAuthMode
	if mode == skeldesc.AuthModeOff {
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

func (o *RpcOperation) loadMethodDescriptor() bool {
	if o.methodDescriptor != nil {
		return true
	}

	method := o.serviceDescriptor.MethodBySkelName(o.MethodName)
	if method == nil {
		o.writeError(ex.NotFound, "rpc method descriptor is not found: "+o.ServiceName+"/"+o.MethodName)
		return false
	}

	o.methodDescriptor = method
	return true
}

func (o *RpcOperation) writeErrorWithReason(code ex.Code, message string, reason string) {
	var options []ex.ErrorOption
	if reason != "" {
		options = append(options, ex.WithReason(reason))
	}
	o.writeError(code, message, options...)
}
