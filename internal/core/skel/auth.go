package skel

import "fmt"

type AuthMode string

const (
	// AuthModeUnset inherits authentication from the enclosing context or uses its default behavior.
	AuthModeUnset    AuthMode = "unset"
	AuthModeRequired AuthMode = "required"
	AuthModeOptional AuthMode = "optional"
	AuthModeGuest    AuthMode = "guest"
	AuthModeOff      AuthMode = "off"
)

// Legacy values remain readable from previously generated contracts.
const (
	// AuthModeAuth is the legacy spelling of AuthModeRequired.
	//
	// Deprecated: Use AuthModeRequired instead.
	AuthModeAuth AuthMode = "auth"
	// AuthModeNoAuth is the legacy anonymous-access mode.
	//
	// Deprecated: Use AuthModeOptional for Rpc or AuthModeOff for Web.
	AuthModeNoAuth AuthMode = "noauth"
)

// ConvertLegacyAuthModes rejects Rpc off and converts historical spellings at schema registration.
// Unset and empty modes retain their inheritance/default semantics, and hashes
// remain the identifiers supplied by the generating compiler.
func ConvertLegacyAuthModes(schema *DomainSchema) {
	// Validate before conversion so invalid contracts are not partially rewritten.
	for _, service := range schema.Services {
		checkRpcServiceAuthModes(service)
	}
	for _, actor := range schema.Actors {
		checkRpcServiceAuthModes(actor.AuthService)
		checkRpcServiceAuthModes(actor.PermService)
		checkRpcMethodAuthMode(actor.AuthMethod, actor.SkelName)
		checkRpcMethodAuthMode(actor.PermMethod, actor.SkelName)
	}
	for _, resource := range schema.Resources {
		checkRpcServiceAuthModes(resource.CheckService)
	}
	for _, service := range schema.Services {
		convertLegacyServiceAuthModes(service)
	}
	for _, web := range schema.Webs {
		convertLegacyAuthMode(&web.AuthMode, AuthModeOff)
	}
	for _, actor := range schema.Actors {
		convertLegacyServiceAuthModes(actor.AuthService)
		convertLegacyServiceAuthModes(actor.PermService)
		if actor.AuthMethod != nil {
			convertLegacyAuthMode(&actor.AuthMethod.AuthMode, AuthModeOptional)
		}
		if actor.PermMethod != nil {
			convertLegacyAuthMode(&actor.PermMethod.AuthMode, AuthModeOptional)
		}
	}
	for _, resource := range schema.Resources {
		convertLegacyServiceAuthModes(resource.CheckService)
	}
}

func convertLegacyServiceAuthModes(service *ServiceSchema) {
	if service == nil {
		return
	}
	convertLegacyAuthMode(&service.AuthMode, AuthModeOptional)
	for _, method := range service.Methods {
		convertLegacyAuthMode(&method.AuthMode, AuthModeOptional)
	}
}

func convertLegacyAuthMode(mode *AuthMode, noAuthMode AuthMode) {
	switch *mode {
	case AuthModeAuth:
		*mode = AuthModeRequired
	case AuthModeNoAuth:
		*mode = noAuthMode
	}
}

func checkRpcServiceAuthModes(service *ServiceSchema) {
	if service == nil {
		return
	}
	if service.AuthMode == AuthModeOff {
		panic(fmt.Errorf("skel: Rpc service %s cannot use auth off; use auth optional for anonymous access", service.SkelName))
	}
	for _, method := range service.Methods {
		checkRpcMethodAuthMode(method, service.SkelName)
	}
}

func checkRpcMethodAuthMode(method *MethodSchema, ownerName string) {
	if method != nil && method.AuthMode == AuthModeOff {
		panic(fmt.Errorf("skel: Rpc method %s/%s cannot use auth off; use auth optional for anonymous access", ownerName, method.SkelName))
	}
}
