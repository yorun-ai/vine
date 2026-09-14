package seeder

import (
	"crypto/sha256"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResolveSeedVariablesAndSources(t *testing.T) {
	template := []byte("appConfigs:\n- name: demo.Config\n  value:\n    enabled: ${enabled}\n    port: ${port}\n    origins: ${origins}\n    url: https://${host}:${port}\n")
	source := []byte(fmt.Sprintf("version: 1\nseedSha256: %x\nfields:\n  /appConfigs/0/value/url:\n    source: profile/dev\n    define: domain/booker\n    override: profile/dev\n", sha256.Sum256(template)))
	node, origins, err := resolveSeedInput(template, []byte("enabled: false\nport: 8080\norigins: [https://example.com]\nhost: example.com\n"), source)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, node.Decode(&value))
	config := value["appConfigs"].([]any)[0].(map[string]any)["value"].(map[string]any)
	require.Equal(t, false, config["enabled"])
	require.Equal(t, 8080, config["port"])
	require.Equal(t, []any{"https://example.com"}, config["origins"])
	require.Equal(t, "https://example.com:8080", config["url"])
	origin := entityFieldSources(origins, "appConfigs", 0)["/value/url"]
	require.Equal(t, "profile/dev", origin.Source)
	require.Equal(t, "domain/booker", origin.Define)
	require.Equal(t, "profile/dev", origin.Override)
	require.Equal(t, []string{"host", "port"}, origin.Variables)
}

func TestSeedVariablesAreLiteralAndDoNotMutateDictionary(t *testing.T) {
	node, _, err := resolveSeedInput([]byte("appConfigs: [{name: demo.Config, value: '${value}'}]"), []byte("value: '${unknown}'\n"), nil)
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.Equal(t, "${unknown}", payload.AppConfigs[0].Value)
}

func TestResolveSeedRejectsInvalidInputs(t *testing.T) {
	for _, tc := range []struct{ name, seed, vars, source string }{
		{"missing variable", "appConfigs: [{name: demo.Config, value: '${missing}'}]", "", ""},
		{"collection interpolation", "appConfigs: [{name: demo.Config, value: 'x${value}'}]", "value: []", ""},
		{"duplicate variable", "{}", "value: 1\nvalue: 2", ""},
		{"multiple documents", "{}\n---\n{}", "", ""},
		{"mapping key", "'${name}': value", "name: field", ""},
		{"bad digest", "{}", "", "version: 1\nseedSha256: bad\nfields: {}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := resolveSeedInput([]byte(tc.seed), []byte(tc.vars), []byte(tc.source))
			require.Error(t, err)
		})
	}
}

func TestConfigVariableSourcesStopAtValueKeys(t *testing.T) {
	template := []byte(`appConfigs:
- name: demo.Config
  value:
    list: ['${b}', '${a}', '${b}']
    object: {first: '${a}', nested: {second: '${b}'}}
`)
	source := []byte(fmt.Sprintf("version: 1\nseedSha256: %x\nfields:\n  /appConfigs/0/value/list: {source: domain/demo, define: domain/demo}\n  /appConfigs/0/value/object: {source: app/default, define: domain/demo, override: app/default}\n", sha256.Sum256(template)))
	_, origins, err := resolveSeedInput(template, []byte("a: one\nb: two\n"), source)
	require.NoError(t, err)
	require.Len(t, origins, 2)
	for _, key := range []string{"list", "object"} {
		require.Equal(t, []string{"a", "b"}, origins["/appConfigs/0/value/"+key].Variables)
		require.Equal(t, "domain/demo", origins["/appConfigs/0/value/"+key].Define)
	}
	require.Equal(t, "app/default", origins["/appConfigs/0/value/object"].Override)
	deeper := []byte(fmt.Sprintf("version: 1\nseedSha256: %x\nfields:\n  /appConfigs/0/value/list/0: {source: domain/demo, define: domain/demo}\n", sha256.Sum256(template)))
	_, _, err = resolveSeedInput(template, []byte("a: one\nb: two\n"), deeper)
	require.ErrorContains(t, err, "invalid seed source field")
}

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

func TestBindingsKeepOriginalNestedTemplateAndEachDefault(t *testing.T) {
	template := []byte(`appConfigs:
- name: demo.Config
  value:
    options:
      first: "${missing:first}"
      second: "${missing:second}"
      actual: "${items}"
      zero: "${zero:42}"
      empty: "${empty:fallback}"
      nullable: "${nil:fallback}"
`)
	_, sources, err := resolveSeedInputWithSchemas(template, []byte("items: [1, false]\nzero: 0\nempty: ''\nnil: null\n"), nil, nil)
	require.NoError(t, err)
	source := sources["/appConfigs/0/value/options"]
	require.JSONEq(t, `{"first":"${missing:first}","second":"${missing:second}","actual":"${items}","zero":"${zero:42}","empty":"${empty:fallback}","nullable":"${nil:fallback}"}`, string(*source.Template))
	require.Len(t, source.Bindings, 6)
	expected := []struct {
		path, value string
		fallback    bool
	}{{"/first", `"first"`, true}, {"/second", `"second"`, true}, {"/actual", `[1,false]`, false}, {"/zero", `0`, false}, {"/empty", `""`, false}, {"/nullable", `null`, false}}
	for i, want := range expected {
		require.Equal(t, want.path, source.Bindings[i].Path)
		require.JSONEq(t, want.value, string(source.Bindings[i].Value))
		require.Equal(t, want.fallback, source.Bindings[i].DefaultUsed)
	}
}

func TestBindingsRecordAppliedWholeObjectAndLiteralInterpolation(t *testing.T) {
	_, sources, err := resolveSeedInputWithSchemas([]byte(`appConfigs: [{name: app.DatabaseConfig, value: '${database}'}]`), []byte(`database: {host: '${literal}', port: 5432, extra: unused}`), nil, testVarsSchemas())
	require.NoError(t, err)
	source := sources["/appConfigs/0/value"]
	require.Equal(t, `"${database}"`, string(*source.Template))
	require.Len(t, source.Bindings, 1)
	require.JSONEq(t, `{"host":"${literal}","port":5432}`, string(source.Bindings[0].Value))
	_, sources, err = resolveSeedInputWithSchemas([]byte(`portalRules: [{name: app.rule, matchHost: 'https://${host}:${port:443}'}]`), []byte("host: example.com"), nil, nil)
	require.NoError(t, err)
	source = sources["/portalRules/0/matchHost"]
	require.Len(t, source.Bindings, 2)
	require.Equal(t, `"https://${host}:${port:443}"`, string(*source.Template))
	require.Equal(t, `"443"`, string(source.Bindings[1].Value))
	require.True(t, source.Bindings[1].DefaultUsed)
}
