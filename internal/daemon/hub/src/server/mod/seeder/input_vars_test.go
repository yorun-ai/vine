package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedAssignmentsOverlayFileAndPreserveTypes(t *testing.T) {
	template := []byte(`appConfigs:
- name: app.DatabaseConfig
  value: '${database}'
portalCerts:
- name: app.cert
  domains: '${origins}'
portalRules:
- name: app.rule
  matchHost: '${text}'
  disabled: '${enabled:true}'
`)
	node, sources, err := resolveSeedInputWithSchemas(template,
		[]byte("database: {host: file, port: 5432}"), nil, testVarsSchemas(),
		"database.host=first", "database.host=last", `origins=["a.com","b.com"]`,
		"enabled=false", "text=https://a.com/?x=a=b,c")
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.JSONEq(t, `{"host":"last","port":5432}`, payload.AppConfigs[0].Value)
	require.Equal(t, []string{"a.com", "b.com"}, payload.PortalCerts[0].Domains)
	require.Equal(t, "https://a.com/?x=a=b,c", payload.PortalRules[0].MatchHost)
	require.False(t, payload.PortalRules[0].Disabled)
	require.Equal(t, []string{"database"}, sources["/appConfigs/0/value"].Variables)
}

func TestSeedAssignmentsReplaceObjectsAndRemainLiteral(t *testing.T) {
	dictionary, err := parseSeedNode([]byte("database: {host: file, port: 5432, stale: true}"))
	require.NoError(t, err)
	dictionary, err = applySeedVariables(dictionary, []string{
		"database={host: localhost, port: 5433}", "database.host=override", `text="123"`, "empty=", "optional=null",
	})
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, dictionary.Decode(&value))
	require.Equal(t, map[string]any{"host": "override", "port": 5433}, value["database"])
	require.Equal(t, "123", value["text"])
	require.Equal(t, "", value["empty"])
	require.Nil(t, value["optional"])

	node, _, err := resolveSeedInputWithSchemas([]byte("portalRules: [{name: app.rule, matchHost: '${text}'}]"), nil, nil,
		testVarsSchemas(), "text=${other}")
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.Equal(t, "${other}", payload.PortalRules[0].MatchHost)
}

func TestSeedAssignmentsRejectMalformedInputs(t *testing.T) {
	for _, assignments := range [][]string{
		{"missingEquals"}, {"=value"}, {"a..b=1"}, {"a_b=1"}, {"a.0=1"},
		{"a=["}, {"a=1\n---\n2"}, {"a={x: 1, x: 2}"}, {"a=&anchor value"},
		{"a=1", "a.b=2"}, {"a=null", "a.b=2"}, {"a=[]", "a.b=2"},
	} {
		t.Run(assignments[0], func(t *testing.T) {
			_, err := applySeedVariables(nil, assignments)
			require.Error(t, err)
		})
	}
}

func TestSeedAssignmentsUseExistingSchemaValidation(t *testing.T) {
	template := []byte("appConfigs: [{name: app.DatabaseConfig, value: '${database}'}]")
	_, _, err := resolveSeedInputWithSchemas(template, nil, nil, testVarsSchemas(),
		"database={host: localhost, port: wrong}")
	require.ErrorContains(t, err, "expected int")
	_, _, err = resolveSeedInputWithSchemas(template, nil, nil, testVarsSchemas(), "database={host: localhost}")
	require.ErrorContains(t, err, "port: required field is missing")
	_, _, err = resolveSeedInputWithSchemas([]byte("{}"), nil, nil, testVarsSchemas(), "unused=wrong")
	require.NoError(t, err)
}
