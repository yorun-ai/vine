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
		domains, _ := decodeRegisteredDescriptors(reg)
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
	domains, _ := decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{valid}})
	require.Equal(t, descriptor.AuthModeRequired, domains[0].Services[0].Methods[0].EffectiveAuthMode)
	for _, raw := range []skeltype.JSON{
		`{"name":"demo","domain":"demo"}`,
		`{"name":"demo","generated":{"compilerVersion":"v99.0.0"},"services":[{"name":"Api","api":true,"authMode":"required","methods":[{"name":"Get","authMode":"inherit","effectiveAuthMode":"optional"}]}]}`,
		`{"domain":"demo","generated":{"compilerVersion":"v99.0.0"},"services":[{"name":"Old","audiences":[{"skelName":"demo.User"}]}]}`,
	} {
		require.Panics(t, func() {
			decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{raw}})
		})
	}
	require.Panics(t, func() {
		decodeRegisteredDescriptors(skeled.AppRegistration{DomainDescriptors: []skeltype.JSON{valid, valid}})
	})
}

func TestRegistrationRestoresMissingHashesFromItsOwnLegacyContracts(t *testing.T) {
	const legacySchema skeltype.JSON = `{"domain":"old","generated":{"compilerVersion":"v0.30.0"},
		"services":[{"skelName":"old.Api","hash":"api"}],
		"actors":[{
		 "authEnabled":true, "authCredential":{}, "authInfo":{}, "authMethod":{"name":"Auth"}, "authService":{"skelName":"old.Auth","hash":"auth","methods":[{"name":"Auth"}]},
		 "permEnabled":true, "permMethod":{"name":"CheckCodes"}, "permService":{"skelName":"old.Perm","hash":"perm","methods":[{"name":"CheckCodes"}]}
		}],
		"resources":[{"checkService":{"skelName":"old.Check","hash":"check"}}],
		"webs":[{"skelName":"old.Web","hash":"web"}],
		"events":[{"skelName":"old.Event","hash":"event"}],
		"tasks":[
		{"skelName":"old.Tick","hash":"legacy-tick"},
		{"skelName":"old.Conflict","hash":"first"},
		{"skelName":"old.Conflict","hash":"second"}
	]}`
	const descriptor skeltype.JSON = `{"name":"current","generated":{"compilerVersion":"v0.35.0"},"tasks":[
		{"skelName":"current.Tick","hash":"current-tick"}
	]}`
	for _, reg := range []skeled.AppRegistration{
		{DomainSchemas: []skeltype.JSON{legacySchema}, DomainDescriptors: []skeltype.JSON{descriptor}},
		{DomainDescriptors: []skeltype.JSON{legacySchema, descriptor}},
	} {
		reg.TaskRunners = []skeled.TaskRunnerRegistration{
			{TaskSkelName: "old.Tick"},
			{TaskSkelName: "old.Tick", DescriptorHash: "explicit"},
			{TaskSkelName: "current.Tick"},
			{TaskSkelName: "old.Conflict"},
		}
		for _, name := range []string{"old.Api", "old.Auth", "old.Perm", "old.Check"} {
			reg.ServiceHandlers = append(reg.ServiceHandlers, skeled.ServiceHandlerRegistration{ServiceSkelName: name})
		}
		reg.WebHandlers = []skeled.WebHandlerRegistration{{WebSkelName: "old.Web"}}
		reg.EventListeners = []skeled.EventListenerRegistration{{EventSkelName: "old.Event"}}
		registration := toCoreAppRegistration(reg)
		for i, hash := range []string{"api", "auth", "perm", "check"} {
			require.Equal(t, hash, registration.ServiceHandlers[i].DescriptorHash)
		}
		require.Equal(t, "web", registration.WebHandlers[0].DescriptorHash)
		require.Equal(t, "event", registration.EventListeners[0].DescriptorHash)
		got := registration.TaskRunners
		require.Len(t, got, 4)
		require.Equal(t, "legacy-tick", got[0].DescriptorHash)
		require.Equal(t, "explicit", got[1].DescriptorHash)
		require.Empty(t, got[2].DescriptorHash)
		require.Empty(t, got[3].DescriptorHash)
	}
}
