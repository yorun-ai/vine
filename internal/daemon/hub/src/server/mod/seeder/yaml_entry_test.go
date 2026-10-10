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
		require.True(t, entry.Http.HttpEnabled)
		require.True(t, entry.Http.HttpsEnabled)
		require.True(t, entry.Http.AutoHTTPS)
		require.Equal(t, 80, entry.Http.HttpPort)
	}
	payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [{name: web, protocol: http, http: {httpsEnabled: false, autoHTTPS: false}}]")
	require.NoError(t, err)
	entry := core.NormalizePortalEntry(*payload.PortalEntries[0].toCorePortalEntry())
	require.False(t, entry.Http.HttpsEnabled)
	require.False(t, entry.Http.AutoHTTPS)
	for _, value := range []string{
		"{protocol: http, scheme: http}", "{protocol: http, scheme: ''}", "{protocol: http, port: 0}", "{protocol: http, port: null}",
		"{scheme: http, http: {}}", "{protocol: null}", "{protocol: http, http: null}", "{protocol: http, http: {autoHTTPS: null}}", "{protocol: http, http: {typo: true}}",
	} {
		_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries: [" + value + "]")
		require.Error(t, err, value)
	}
}
