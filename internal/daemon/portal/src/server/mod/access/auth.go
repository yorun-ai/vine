package access

import (
	"bytes"
	"encoding/json/jsontext"
	"net/http"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/mtls"
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
	identity        *mtls.Identity

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
	info, code, message, ok := o.invoke[jsontext.Value](authRequest, skelServiceName, "auth", "auth failed")
	if !ok {
		writeError(code, message)
		return false
	}

	if len(info) == 0 || bytes.Equal(info, []byte("null")) {
		writeError(ex.ServiceUnavailable, "bad auth response")
		return false
	}

	setActor(meta.NewAuthenticatedActorWithRawInfo(o.actorSchema.AuthInfo.SkelName, info))
	return true
}
