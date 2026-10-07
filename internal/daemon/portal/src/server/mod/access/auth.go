package access

import (
	"bytes"
	"encoding/json/jsontext"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vpre"
)

const headerAuthorization = "Authorization"

type _AuthErrorWriter func(ex.Code, string, ...ex.ErrorOption)
type _AuthActorSetter func(meta.Actor)

type Auther struct {
	Request  *http.Request
	Response http.ResponseWriter

	Trace     meta.Trace
	Initiator meta.Initiator

	endpointManager *epmgr.Manager
	actorDescriptor *skeldesc.Actor
	identity        *mtls.Identity

	actor      meta.Actor
	credential map[string]string
}

func (o *Auther) auth(writeError _AuthErrorWriter, setActor _AuthActorSetter) bool {
	vpre.CheckNotNil(o.actorDescriptor.Auth.Credential, "actor auth credential descriptor is not configured")
	vpre.CheckNotNil(o.actorDescriptor.Auth.Info, "actor auth info descriptor is not configured")

	if !o.parseCredential(writeError) {
		return false
	}

	o.actor = meta.NewAuthenticatingActor()
	authRequest := o.buildInvokeRequest(
		o.actorDescriptor.Auth.Service.SkelName,
		o.actorDescriptor.Auth.Method().SkelName,
		map[string]any{"credential": o.credential},
	)
	if !o.executeAuthRequest(authRequest, writeError, setActor) {
		return false
	}

	return true
}

func (o *Auther) parseCredential(writeError _AuthErrorWriter) bool {
	values := o.Request.Header.Values(headerAuthorization)
	if len(values) != 1 {
		writeError(ex.Unauthorized, "bad credential: expected one Authorization header")
		return false
	}
	authorization := values[0]
	credential, ok := parseCredential(o.actorDescriptor.Auth.Credential, authorization)
	if !ok {
		writeError(ex.Unauthorized, "bad credential")
		return false
	}

	o.credential = credential
	return true
}

func parseCredential(descriptor *skeldesc.Data, authorization string) (map[string]string, bool) {
	credentialNames := map[string]string{}
	for _, member := range descriptor.Members {
		credentialNames[strings.ToLower(member.Name)] = member.Name
	}
	// skelc rejects empty actor credentials; keep this guard for stale descriptor data.
	if len(credentialNames) == 0 {
		return nil, false
	}

	credential := map[string]string{}
	for part := range strings.SplitSeq(authorization, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key, value, ok := strings.Cut(part, " ")
		if !ok {
			return nil, false
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		name, ok := credentialNames[strings.ToLower(key)]
		if !ok || value == "" {
			return nil, false
		}
		credential[name] = value
	}

	for _, member := range descriptor.Members {
		if member.Type != nil && member.Type.Nullable {
			continue
		}
		if _, present := credential[member.Name]; !present {
			return nil, false
		}
	}
	// skelc guarantees at least one required credential field.
	return credential, true
}

func (o *Auther) executeAuthRequest(authRequest *http.Request, writeError _AuthErrorWriter, setActor _AuthActorSetter) bool {
	skelServiceName := o.actorDescriptor.Auth.Service.SkelName
	info, code, message, reason, ok := o.invoke[jsontext.Value](authRequest, skelServiceName, "auth", "auth failed")
	if !ok {
		var options []ex.ErrorOption
		if reason != "" {
			options = append(options, ex.WithReason(reason))
		}
		writeError(code, message, options...)
		return false
	}

	if len(info) == 0 || bytes.Equal(info, []byte("null")) {
		writeError(ex.ServiceUnavailable, "bad auth response")
		return false
	}

	identifier := ""
	if o.actorDescriptor.Auth.IdentifierField != "" {
		value := gjson.GetBytes(info, o.actorDescriptor.Auth.IdentifierField)
		if value.Type == gjson.Null {
			writeError(ex.ServiceUnavailable, "bad auth response")
			return false
		}
		identifier = value.Raw
		if value.Type == gjson.String {
			identifier = value.Str
		}
	}
	setActor(meta.NewAuthenticatedActorWithRawInfo(o.actorDescriptor.SkelName, identifier, o.actorDescriptor.Auth.Info.SkelName, info))
	return true
}

// authenticate applies explicit modes without treating failed authentication as anonymous.
func (o *Auther) authenticate(mode skeldesc.AuthMode, writeError _AuthErrorWriter, setActor _AuthActorSetter) bool {
	switch mode {
	case skeldesc.AuthModeOff:
		setActor(meta.NewAnonymousActor())
		return true
	case skeldesc.AuthModeOptional, skeldesc.AuthModeAnonymous:
		if len(o.Request.Header.Values(headerAuthorization)) == 0 {
			setActor(meta.NewAnonymousActor())
			o.Request.Header.Del(headerAuthorization)
			return true
		}
	case skeldesc.AuthModeRequired:
	default:
		writeError(ex.ServiceUnavailable, "unsupported auth mode")
		return false
	}
	if o.actorDescriptor.Auth == nil {
		writeError(ex.ClientForbidden, "endpoint requires auth, but actor auth not enabled")
		return false
	}
	if !o.auth(writeError, setActor) {
		return false
	}
	if mode == skeldesc.AuthModeAnonymous {
		writeError(ex.ClientForbidden, "endpoint only allows anonymous access")
		return false
	}
	// Portal forwards the admitted actor without the client's credentials.
	o.Request.Header.Del(headerAuthorization)
	return true
}
