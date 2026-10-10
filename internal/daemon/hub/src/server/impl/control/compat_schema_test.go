package control

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
)

func TestRegistrationConvertsLegacySchemaAtWireBoundary(t *testing.T) {
	raw := skeltype.JSON(`{"domain":"demo","hash":"unchanged","generated":{"compilerVersion":"v0.17.1"},"services":[{"name":"Api","skelName":"demo.Api","api":true,"authMode":"required","require":{"expr":{"mode":"code","code":"read"}},"methods":[{"name":"Get","skelName":"get","authMode":"optional"}]}]}`)
	for _, reg := range []skeled.AppRegistration{{DomainSchemas: []skeltype.JSON{raw}}, {DomainDescriptors: []skeltype.JSON{raw}}} {
		domains := decodeRegisteredDescriptors(reg)
		require.Len(t, domains, 1)
		require.Equal(t, "unchanged", domains[0].Hash)
		method := domains[0].Services[0].Methods[0]
		require.Equal(t, descriptor.AuthModeOptional, method.EffectiveAuthMode)
		require.Equal(t, "read", method.EffectiveRequire.Expression.Code)
		require.NoError(t, descriptor.ValidateEffectivePolicy(domains[0]))
	}
}

func TestRegistrationValidatesNewDescriptorWithoutRepair(t *testing.T) {
	valid := skeltype.JSON(`{"name":"demo","generated":{"compilerVersion":"v99.0.0"},"services":[{"name":"Api","api":true,"authMode":"required","methods":[{"name":"Get","skelName":"get","authMode":"inherit","effectiveAuthMode":"required"}]}]}`)
	domains := decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{valid}})
	require.Equal(t, descriptor.AuthModeRequired, domains[0].Services[0].Methods[0].EffectiveAuthMode)
	for _, raw := range []skeltype.JSON{
		`{"name":"demo","domain":"demo"}`,
		`{"name":"demo","generated":{"compilerVersion":"v99.0.0"},"services":[{"name":"Api","api":true,"authMode":"required","methods":[{"name":"Get","authMode":"inherit","effectiveAuthMode":"optional"}]}]}`,
		`{"domain":"demo","generated":{"compilerVersion":"v99.0.0"},"services":[{"name":"Old","audiences":[{"skelName":"demo.User"}]}]}`,
	} {
		require.Panics(t, func() { decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{raw}}) })
	}
	require.Panics(t, func() {
		decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{valid, valid}})
	})
}
