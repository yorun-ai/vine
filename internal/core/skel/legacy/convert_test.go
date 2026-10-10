package legacy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
)

func TestConvertNormalizesAuthAndComputesPolicy(t *testing.T) {
	for _, mode := range []AuthMode{"", "required", "optional"} {
		t.Run(string(mode), func(t *testing.T) {
			source := &DomainSchema{
				Domain: "demo",
				Hash:   "original",
				Services: []*ServiceSchema{{
					Name:     "Api",
					SkelName: "demo.Api",
					Api:      true,
					AuthMode: mode,
					Require: &PermRequire{
						Expr: &PermExpr{
							Mode: PermRequireModeCode,
							Code: "read",
						},
					},
					Methods: []*MethodSchema{
						{
							Name:     "Get",
							SkelName: "get",
						},
						{
							Name:     "List",
							SkelName: "list",
							AuthMode: AuthMode("optional"),
						},
					},
				}},
				Webs: []*WebSchema{{
					AuthMode: mode,
				}},
				Configs: []*ConfigSchema{{
					Lifecycle: "ETERNAL",
				}},
			}
			result, err := Convert(source)
			require.NoError(t, err)
			expected := descriptor.AuthModeRequired
			if mode == AuthMode("optional") {
				expected = descriptor.AuthModeOptional
			}
			require.Equal(t, expected, result.Services[0].AuthMode)
			require.Equal(t, descriptor.AuthModeInherit, result.Services[0].Methods[0].AuthMode)
			require.Equal(t, expected, result.Services[0].Methods[0].EffectiveAuthMode)
			require.Equal(t, descriptor.AuthModeOptional, result.Services[0].Methods[1].EffectiveAuthMode)
			require.Equal(t, "read", result.Services[0].Methods[0].EffectiveRequire.Expression.Code)
			require.NoError(t, descriptor.ValidateEffectivePolicy(result))

			require.Equal(t, expected, result.Webs[0].AuthMode)
			require.Equal(t, descriptor.ConfigLifecycleEternal, result.Configs[0].Lifecycle)
			require.Equal(t, "original", result.Hash)
			require.Equal(t, mode, source.Services[0].AuthMode)
			require.Empty(t, source.Services[0].Methods[0].AuthMode)
		})
	}
}

func TestConvertCallbackReferences(t *testing.T) {
	method := &MethodSchema{
		Name:     "Authenticate",
		SkelName: "authenticate",
		AuthMode: AuthMode("optional"),
	}
	service := &ServiceSchema{
		Name:     "Auth",
		SkelName: "demo.Auth",
		Methods:  []*MethodSchema{method},
	}
	actor := &ActorSchema{
		Name:           "User",
		SkelName:       "demo.User",
		AuthEnabled:    true,
		AuthCredential: &DataSchema{},
		AuthInfo:       &DataSchema{},
		AuthService:    service,
		AuthMethod:     method,
		PermEnabled:    true,
		PermService:    service,
		PermMethod:     method,
	}
	result, err := Convert(&DomainSchema{
		Domain: "demo",
		Actors: []*ActorSchema{actor},
		Resources: []*ResourceSchema{{
			Name:         "Item",
			CheckService: service,
			Checks: []*ResourceCheckSchema{{
				Name:   "read",
				Method: method,
			}},
		}},
	})
	require.NoError(t, err)
	require.Same(t, result.Actors[0].Auth.Service.Methods[0], result.Actors[0].Auth.Method())
	require.Same(t, result.Actors[0].Permission.Service.Methods[0], result.Actors[0].Permission.Method())
	require.Same(t, result.Resources[0].CheckService.Methods[0], result.Resources[0].CheckMethod(result.Resources[0].Checks[0]))
	require.Equal(t, descriptor.AuthModeOptional, result.Actors[0].Auth.Method().EffectiveAuthMode)
}

func TestConvertUnmarkedService(t *testing.T) {
	result, err := Convert(&DomainSchema{
		Domain:   "demo",
		Services: []*ServiceSchema{{}},
	})
	require.NoError(t, err)
	service := result.Services[0]
	require.False(t, service.Api)
	require.False(t, service.Pub)
}

func TestConvertRejectsRetiredAuthenticationModes(t *testing.T) {
	for _, mode := range []AuthMode{"auth", "noauth", "unset"} {
		_, err := Convert(&DomainSchema{Domain: "demo", Services: []*ServiceSchema{{Name: "Api", SkelName: "demo.Api", Api: true, AuthMode: mode, Methods: []*MethodSchema{{Name: "Get"}}}}})
		require.Error(t, err, mode)
	}
}
