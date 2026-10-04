package skel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterDomainSchemaConvertsLegacyAuthModes(t *testing.T) {
	schema := new(DomainSchema{
		Domain: "demo", Hash: "original", Generated: validGeneratedInfoForTest(),
		Services: []*ServiceSchema{{Hash: "service", AuthMode: AuthModeAuth, Methods: []*MethodSchema{
			{Hash: "method", AuthMode: AuthModeNoAuth}, {AuthMode: AuthModeUnset}, {AuthMode: ""}, {AuthMode: AuthModeOptional}, {AuthMode: AuthModeGuest}, {AuthMode: "unknown"},
		}}},
		Webs: []*WebSchema{{AuthMode: AuthModeNoAuth}, {AuthMode: AuthModeAuth}, {AuthMode: ""}},
		Actors: []*ActorSchema{{
			AuthService: new(ServiceSchema{AuthMode: AuthModeAuth}),
			AuthMethod:  new(MethodSchema{AuthMode: AuthModeNoAuth}),
			PermService: new(ServiceSchema{AuthMode: AuthModeNoAuth}),
			PermMethod:  new(MethodSchema{AuthMode: AuthModeAuth}),
		}},
		Resources: []*ResourceSchema{{CheckService: new(ServiceSchema{AuthMode: AuthModeAuth})}},
	})
	registry := NewRegistry()
	registry.RegisterDomainSchema(schema)
	registered := registry.RegisteredDomainSchemas()[0]
	require.Equal(t, AuthModeRequired, registered.Services[0].AuthMode)
	require.Equal(t, []AuthMode{AuthModeOptional, AuthModeUnset, "", AuthModeOptional, AuthModeGuest, "unknown"}, []AuthMode{
		registered.Services[0].Methods[0].AuthMode, registered.Services[0].Methods[1].AuthMode,
		registered.Services[0].Methods[2].AuthMode, registered.Services[0].Methods[3].AuthMode,
		registered.Services[0].Methods[4].AuthMode, registered.Services[0].Methods[5].AuthMode,
	})
	require.Equal(t, AuthModeOff, registered.Webs[0].AuthMode)
	require.Equal(t, AuthModeRequired, registered.Webs[1].AuthMode)
	require.Empty(t, registered.Webs[2].AuthMode)
	require.Equal(t, AuthModeRequired, registered.Actors[0].AuthService.AuthMode)
	require.Equal(t, AuthModeOptional, registered.Actors[0].AuthMethod.AuthMode)
	require.Equal(t, AuthModeOptional, registered.Actors[0].PermService.AuthMode)
	require.Equal(t, AuthModeRequired, registered.Actors[0].PermMethod.AuthMode)
	require.Equal(t, AuthModeRequired, registered.Resources[0].CheckService.AuthMode)
	require.Equal(t, "original", registered.Hash)
	require.Equal(t, "service", registered.Services[0].Hash)
	require.Equal(t, "method", registered.Services[0].Methods[0].Hash)
}

func TestRegisterFullSchemaConvertsLegacyAuthModes(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterDomainSchema(new(DomainSchema{Domain: "demo", Generated: validGeneratedInfoForTest()}))
	schema := new(DomainSchema{Domain: "demo", Full: true, Generated: validGeneratedInfoForTest(), Services: []*ServiceSchema{{AuthMode: AuthModeNoAuth}}})
	registry.RegisterDomainSchema(schema)
	require.Equal(t, AuthModeOptional, registry.RegisteredDomainSchemas()[0].Services[0].AuthMode)
}

func TestRegisterDomainSchemaRejectsRpcOff(t *testing.T) {
	for _, full := range []bool{false, true} {
		for _, method := range []bool{false, true} {
			t.Run(fmt.Sprintf("full=%t/method=%t", full, method), func(t *testing.T) {
				registry := NewRegistry()
				if full {
					registry.RegisterDomainSchema(new(DomainSchema{Domain: "demo", Generated: validGeneratedInfoForTest()}))
				}
				service := new(ServiceSchema{SkelName: "demo.Api", AuthMode: AuthModeOff})
				name := "Rpc service demo.Api"
				if method {
					service.AuthMode = AuthModeAuth
					service.Methods = []*MethodSchema{{SkelName: "get", AuthMode: AuthModeOff}}
					name = "Rpc method demo.Api/get"
				}
				schema := new(DomainSchema{Domain: "demo", Full: full, Generated: validGeneratedInfoForTest(), Services: []*ServiceSchema{service}})
				require.PanicsWithError(t, "skel: "+name+" cannot use auth off; use auth optional for anonymous access", func() {
					registry.RegisterDomainSchema(schema)
				})
				if method {
					require.Equal(t, AuthModeAuth, service.AuthMode, "validation must precede conversion")
				}
				if full {
					require.False(t, registry.RegisteredDomainSchemas()[0].Full)
				} else {
					require.Empty(t, registry.RegisteredDomainSchemas())
				}
			})
		}
	}
}

func TestConvertLegacyAuthModesRejectsEmbeddedRpcOff(t *testing.T) {
	for _, reference := range []string{"auth service", "perm service", "auth method", "perm method", "resource check"} {
		t.Run(reference, func(t *testing.T) {
			actor := new(ActorSchema{SkelName: "demo.User"})
			resource := new(ResourceSchema{})
			schema := new(DomainSchema{Actors: []*ActorSchema{actor}, Resources: []*ResourceSchema{resource}, Webs: []*WebSchema{{AuthMode: AuthModeOff}}})
			service := new(ServiceSchema{SkelName: "demo.Auth", AuthMode: AuthModeOff})
			method := new(MethodSchema{SkelName: "authenticate", AuthMode: AuthModeOff})
			switch reference {
			case "auth service":
				actor.AuthService = service
			case "perm service":
				actor.PermService = service
			case "auth method":
				actor.AuthMethod = method
			case "perm method":
				actor.PermMethod = method
			case "resource check":
				resource.CheckService = service
			}
			require.Panics(t, func() { ConvertLegacyAuthModes(schema) })
		})
	}
	schema := new(DomainSchema{Webs: []*WebSchema{{AuthMode: AuthModeOff}}})
	require.NotPanics(t, func() { ConvertLegacyAuthModes(schema) })
	require.Equal(t, AuthModeOff, schema.Webs[0].AuthMode)
}
