package seeder

import (
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"testing"
)

func TestSeedHTTPBlockDefaultsAndPresence(t *testing.T) {
	for _, value := range []string{"{name: web, protocol: http}", "{name: web, protocol: http, http: {}}", "{name: web, protocol: http, http: {httpsPort: 8443}}"} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + value + "]")
		require.NoError(t, err)
		entry := core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry())
		require.EqualValues(t, "AUTO", entry.Http.Http)
		require.EqualValues(t, "AUTO", entry.Http.Https)
		require.EqualValues(t, "AUTO", entry.Http.AutoHTTPS)
		require.Equal(t, 80, entry.Http.Effective(false).HttpPort)
	}
	payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [{name: web, protocol: http, http: {https: DISABLED, autohttps: DISABLED}}]")
	require.NoError(t, err)
	entry := core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry())
	require.EqualValues(t, "DISABLED", entry.Http.Https)
	require.False(t, entry.Http.Effective(false).AutoHTTPSEnabled)
	for _, value := range []string{
		"{protocol: http, scheme: http}", "{protocol: http, scheme: ''}", "{protocol: http, port: 0}", "{protocol: http, port: null}",
		"{scheme: http, http: {}}", "{protocol: null}", "{protocol: http, http: null}", "{protocol: http, http: {autohttps: null}}", "{protocol: http, http: {typo: true}}",
	} {
		_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + value + "]")
		require.Error(t, err, value)
	}
}

func TestSeedLegacyEntriesPreserveTransportAndPort(t *testing.T) {
	for _, fixture := range []struct {
		yaml string
		http bool
		port int
	}{
		{"{name: web, scheme: http}", true, 80},
		{"{name: web, scheme: http, port: 7088}", true, 7088},
		{"{name: web, scheme: https, host: example.com, port: 8443}", false, 8443},
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + fixture.yaml + "]")
		require.NoError(t, err)
		entry := core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry())
		require.Equal(t, "http", entry.Protocol)
		require.Equal(t, fixture.http, entry.Http.Http == core.PortalSwitchModeEnabled)
		require.Equal(t, !fixture.http, entry.Http.Https == core.PortalSwitchModeEnabled)
		require.False(t, entry.Http.Effective(false).AutoHTTPSEnabled)
		require.Len(t, entry.ReservedAccesses(), 1)
		require.Equal(t, fixture.port, entry.ReservedAccesses()[0].Port)
	}
}

func TestSeedHTTPHostAndLegacyConversion(t *testing.T) {
	for _, input := range []string{
		"{name: web, protocol: http, http: {host: '*.EXAMPLE.com'}}",
		"{name: web, scheme: https, host: '*.EXAMPLE.com'}",
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + input + "]")
		require.NoError(t, err)
		entry := core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry())
		require.Equal(t, "*.example.com", entry.Http.Host)
	}
	for _, input := range []string{
		"{name: web, protocol: http, host: 'demo.local'}",
		"{name: web, protocol: http, host: ''}",
		"{name: web, protocol: http, host: null}",
		"{name: web, protocol: http, host: '', http: {host: 'demo.local'}}",
		"{name: web, protocol: http, http: {host: null}}",
	} {
		_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + input + "]")
		require.Error(t, err)
	}
}

func TestSeedRejectsHTTPSOnlyWithoutHost(t *testing.T) {
	for _, input := range []string{
		"{name: web, scheme: https}",
		"{name: web, protocol: http, http: {http: DISABLED, https: ENABLED}}",
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + input + "]")
		require.NoError(t, err)
		require.Panics(t, func() { core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry()) })
	}
}

func TestSeedHTTPModesRejectInvalidValues(t *testing.T) {
	for _, field := range []string{"http", "https", "autohttps"} {
		for _, value := range []string{"null", "true", "false", "1", "''", "enabled", "ON", "UNSPECIFIED", "[]", "{}"} {
			_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [{name: web, protocol: http, http: {" + field + ": " + value + "}}]")
			require.Error(t, err, "%s=%s", field, value)
		}
		for _, value := range []string{"ENABLED", "DISABLED", "AUTO"} {
			_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [{name: web, protocol: http, http: {" + field + ": " + value + "}}]")
			require.NoError(t, err, "%s=%s", field, value)
		}
	}
}
