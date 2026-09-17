package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
