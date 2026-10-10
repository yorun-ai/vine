package seeder

import (
	"go.yorun.ai/vine/util/vfile"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalRuleSeedMapsWithoutDomainValidation(t *testing.T) {
	seed := _PortalRule{Name: "mapped", RouteType: "SITE", RoutePathPrefix: "/internal/"}
	rule := seed.toCorePortalRule()
	assert.Zero(t, rule.Id)
	assert.Equal(t, "/internal/", rule.RoutePathPrefix)
	seed.RoutePathPrefix = ""
	assert.Empty(t, seed.toCorePortalRule().RoutePathPrefix)
	seed.RouteType = "PERMANENT_REDIRECT"
	seed.RoutePathPrefix = "/internal"
	assert.NotPanics(t, func() { seed.toCorePortalRule() })
}

func TestSeedPortalRuleFieldNames(t *testing.T) {
	for _, content := range []string{
		"portalRules:\n  - name: example\n    matchScheme: http\n    routeType: SITE\n    routePathPrefix: /internal",
	} {
		payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
		require.NoError(t, err)
		seedRule := payload.PortalRules[0].toSeedRule()
		assert.Equal(t, "http", seedRule.Entry.Scheme)
		assert.Equal(t, "/internal", seedRule.Rule.RoutePathPrefix)
	}
	_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalRules:\n  - scheme: http\n    routeType: SITE")
	require.ErrorContains(t, err, "unknown field")
}

func TestSeedRejectsYAMLReferencesInFilesAndInline(t *testing.T) {
	for _, content := range []string{
		"&seed {appConfigs: []}",
		"ignored: &unused text",
		"<<: {appConfigs: []}",
		"appConfigs: [{name: demo.Config, value: &value {enabled: true}}]",
		"portalSites: [{name: site, urls: &urls []}]",
		"portalRules: [{name: rule, <<: {routeType: SITE}}]",
		"portalCerts: [{name: &name cert, cert: *name}]",
	} {
		t.Run(content, func(t *testing.T) {
			_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
			require.ErrorContains(t, err, "not supported")
			require.ErrorContains(t, err, "line ")
			path := filepath.Join(t.TempDir(), "seed.yaml")
			require.NoError(t, os.WriteFile(path, []byte(content), 0600))
			_, err = vfile.ReadAsYaml[*_SettingsYAMLPayload](path)
			require.ErrorContains(t, err, "not supported")
		})
	}
}

func TestSeedRejectsAmbiguousIntegerValues(t *testing.T) {
	for _, content := range []string{
		"appConfigs: [{name: demo.Config, value: {count: 012}}]",
		"portalRules: [{name: rule, matchPort: 07099}]",
		"unknown: [-012, +012, 0_12]",
	} {
		_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
		require.ErrorContains(t, err, "unsupported YAML number")
	}
	payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("appConfigs: [{name: demo.Config, value: {count: 10, plain: 12, text: '012', keys: {'012': value}}}]")
	require.NoError(t, err)
	require.JSONEq(t, `{"count":10,"plain":12,"text":"012","keys":{"012":"value"}}`, payload.AppConfigs[0].Value)
}

func TestSeedRejectsNonDecimalNumbers(t *testing.T) {
	for _, value := range []string{"1_000", "0b10", "0o12", "0x10", "1e3", "1E-3", ".5", "1."} {
		for _, content := range []string{"value: " + value, value + ": value"} {
			_, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](content)
			require.ErrorContains(t, err, "unsupported YAML number")
		}
	}
}

func TestSeedPortalEntryListenIPs(t *testing.T) {
	payload, err := vcode.UnmarshalYamlS[*_SettingsYAMLPayload](`portalEntries:
  - name: local
    protocol: http
    http: {httpEnabled: true, httpsEnabled: false, autoHTTPS: false}
    listenIPs: ["127.0.0.1", "::1"]
`)
	require.NoError(t, err)
	require.Len(t, payload.PortalEntries, 1)
	assert.Equal(t, []string{"127.0.0.1", "::1"}, payload.PortalEntries[0].toCorePortalEntry().ListenIPs)
	_, err = vcode.UnmarshalYamlS[*_SettingsYAMLPayload]("portalEntries:\n  - name: local\n    scheme: http\n    listenIPs: localhost\n")
	require.Error(t, err)
}
