package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedLocationUsesEntityNames(t *testing.T) {
	root, err := parseSeedNode([]byte(`appConfigs: [{name: app.DatabaseConfig, value: {port: 5432}}]
portalRules: [{name: app.rule}]
portalSites: [{name: app.site}]
portalCerts: [{name: app.cert}]
`))
	require.NoError(t, err)
	for path, want := range map[string]string{
		"/appConfigs/0/value/port": `appConfig "app.DatabaseConfig" field "port"`,
		"/portalRules/0/matchPort": `portalRule "app.rule" field "matchPort"`,
		"/portalSites/0/cors/mode": `portalSite "app.site" field "cors.mode"`,
		"/portalCerts/0/domains/0": `portalCert "app.cert" field "domains.0"`,
	} {
		require.Equal(t, want, seedLocation(root, path))
	}
}
