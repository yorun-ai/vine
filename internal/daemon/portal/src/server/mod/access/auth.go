package access

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vpre"
)

const headerAuthorization = "Authorization"

type _AuthErrorWriter func(ex.Code, string)
type _AuthActorSetter func(meta.Actor)

type Auther struct {
	Request  *http.Request
	Response http.ResponseWriter

	Trace     meta.Trace
	Initiator meta.Initiator

	endpointManager *epmgr.Manager
	actorSchema     *skel.ActorSchema

	actor      meta.Actor
	credential map[string]string
}

func (o *Auther) auth(writeError _AuthErrorWriter, setActor _AuthActorSetter) bool {
	vpre.CheckNotNil(o.actorSchema.AuthCredential, "actor auth credential schema is not configured")
	vpre.CheckNotNil(o.actorSchema.AuthInfo, "actor auth info schema is not configured")

	if !o.parseCredential(writeError) {
		return false
	}

	o.actor = meta.NewAuthenticatingActor()
	authRequest := o.buildInvokeRequest(
		o.actorSchema.AuthService.SkelName,
		o.actorSchema.AuthMethod.SkelName,
		map[string]any{"credential": o.credential},
	)
	if !o.executeAuthRequest(authRequest, writeError, setActor) {
		return false
	}

	return true
}

func (o *Auther) parseCredential(writeError _AuthErrorWriter) bool {
	authorization := o.Request.Header.Get(headerAuthorization)
	credential, ok := parseCredential(o.actorSchema.AuthCredential, authorization)
	if !ok {
		writeError(ex.Unauthorized, "bad credential")
		return false
	}

	o.credential = credential
	return true
}

func parseCredential(schema *skel.DataSchema, authorization string) (map[string]string, bool) {
	credentialNames := map[string]string{}
	for _, member := range schema.Members {
		credentialNames[strings.ToLower(member.Name)] = member.Name
	}
	// skelc rejects empty actor credentials; keep this guard for stale schema data.
	if len(credentialNames) == 0 {
		return nil, false
	}

	credential := map[string]string{}
	hasValue := false
	for _, part := range strings.Split(authorization, ",") {
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
		if !ok {
			return nil, false
		}
		credential[name] = value
		if value != "" {
			hasValue = true
		}
	}

	allKeyPresent := len(credential) == len(credentialNames)
	return credential, allKeyPresent && hasValue
}

func (o *Auther) executeAuthRequest(authRequest *http.Request, writeError _AuthErrorWriter, setActor _AuthActorSetter) bool {
	skelServiceName := o.actorSchema.AuthService.SkelName
	response, code, message, ok := o.forwardInvokeRequest(authRequest, skelServiceName, "auth service")
	if !ok {
		writeError(code, message)
		return false
	}

	defer func() { _ = response.Body.Close() }()
	info, ok := o.parseAuthResponse(response, writeError)
	if !ok {
		return false
	}

	setActor(meta.NewAuthenticatedActorWithRawInfo(o.actorSchema.AuthInfo.SkelName, info))
	return true
}

func (o *Auther) parseAuthResponse(response *http.Response, writeError _AuthErrorWriter) (json.RawMessage, bool) {
	info, code, message, ok := readInvokeResponse[json.RawMessage](response, "auth", "bad auth response", "auth failed")
	if !ok {
		writeError(code, message)
		return nil, false
	}

	if len(info) == 0 || bytes.Equal(info, []byte("null")) {
		writeError(ex.ServiceUnavailable, "bad auth response")
		return nil, false
	}

	return info, true
}
