package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"gopkg.in/yaml.v3"
)

func testVarsSchemas() []*skel.DomainSchema {
	nullable := seedScalar(skel.ScalarString)
	nullable.Nullable = true
	database := []*skel.MemberSchema{{Name: "host", Type: seedScalar(skel.ScalarString)}, {Name: "port", Type: seedScalar(skel.ScalarInt)}}
	return []*skel.DomainSchema{{Domain: "app", Data: []*skel.DataSchema{
		{Name: "Vars", SkelName: "app.Vars", Members: []*skel.MemberSchema{
			{Name: "database", Type: &skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "app.DatabaseVars"}},
			{Name: "enabled", Type: seedScalar(skel.ScalarBool)},
			{Name: "text", Type: seedScalar(skel.ScalarString)},
			{Name: "optional", Type: nullable},
			{Name: "origins", Type: &skel.TypeSchema{Kind: skel.TypeKindList, Element: seedScalar(skel.ScalarString)}},
			{Name: "unused", Type: seedScalar(skel.ScalarInt)},
		}},
		{Name: "DatabaseVars", SkelName: "app.DatabaseVars", Members: append(append([]*skel.MemberSchema{}, database...), &skel.MemberSchema{Name: "oldField", Type: seedScalar(skel.ScalarString)})},
	}, Configs: []*skel.ConfigSchema{
		{SkelName: "app.DatabaseConfig", Members: database},
		{SkelName: "app.OptionalConfig", Members: []*skel.MemberSchema{{Name: "value", Type: nullable}}},
	}}}
}

func TestVarsLegacyRuleUsesCanonicalTargetType(t *testing.T) {
	for _, field := range []string{"port", "matchPort"} {
		t.Run(field, func(t *testing.T) {
			_, _, err := resolveSeedInputWithSchemas([]byte("portalRules: [{name: app.rule, "+field+": '${port}'}]"), []byte("port: null"), nil, nil)
			require.ErrorContains(t, err, "null is not allowed")
		})
	}
}

func TestVarsSchemaOnlyChecksReferencedValues(t *testing.T) {
	template := []byte(`appConfigs:
- name: app.DatabaseConfig
  value:
    host: "${database.host:localhost}"
    port: "${database.port:5432}"
`)
	node, sources, err := resolveSeedInputWithSchemas(template, []byte("unused: wrong-type-but-unused\nextra: ignored\n"), nil, testVarsSchemas())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.JSONEq(t, `{"host":"localhost","port":5432}`, payload.AppConfigs[0].Value)
	require.Equal(t, []string{"database.port"}, sources["/appConfigs/0/value/port"].Variables)
}

func TestVarsWholeObjectUsesTargetRequiredFields(t *testing.T) {
	template := []byte("appConfigs: [{name: app.DatabaseConfig, value: '${database}'}]")
	node, _, err := resolveSeedInputWithSchemas(template, []byte("database: {host: localhost, port: 5432, unknown: ignored}"), nil, testVarsSchemas())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.JSONEq(t, `{"host":"localhost","port":5432}`, payload.AppConfigs[0].Value)
	// Stale Vars fields are not required, but target fields still are.
	_, _, err = resolveSeedInputWithSchemas(template, []byte("database: {host: localhost}"), nil, testVarsSchemas())
	require.ErrorContains(t, err, "port: required field is missing")
}

func TestVarsDefaultsDoNotReplacePresentValues(t *testing.T) {
	template := []byte(`appConfigs:
- name: app.OptionalConfig
  value: {value: "${optional:fallback}"}
- name: app.BoolConfig
  value: {enabled: "${enabled:true}"}
portalRules:
- name: app.rule
  matchHost: "${text:fallback}"
  matchPort: "${database.port:5432}"
portalCerts:
- name: app.cert
  domains: "${origins:[example.com]}"
`)
	node, _, err := resolveSeedInputWithSchemas(template, []byte("optional: null\ntext: ''\ndatabase: {port: 0}\nenabled: false\norigins: []"), nil, testVarsSchemas())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.JSONEq(t, `{"value":null}`, payload.AppConfigs[0].Value)
	require.Equal(t, "", payload.PortalRules[0].MatchHost)
	require.Zero(t, payload.PortalRules[0].MatchPort)
	require.JSONEq(t, `{"enabled":false}`, payload.AppConfigs[1].Value)
	require.Empty(t, payload.PortalCerts[0].Domains)
}

func TestVarsReferencesRejectErrorsAtApplicationPoint(t *testing.T) {
	cases := []struct{ name, reference, variables, want string }{
		{"missing", "${database.port}", "", `variable "database.port" is missing and has no default`},
		{"present null", "${database.port:5432}", "database: {port: null}", "null is not allowed"},
		{"parent null", "${database.port:5432}", "database: null", "parent is not an object"},
		{"parent scalar", "${database.port:5432}", "database: nope", "parent is not an object"},
		{"wrong type", "${database.port:5432}", "database: {port: secret-value}", "expected int"},
		{"invalid default", "${database.port:secret-value}", "", "expected int"},
		{"undeclared", "${database.missing:5432}", "", "not defined in app.DatabaseVars"},
		{"bad path", "${database_port:5432}", "", "camelCase"},
		{"empty path", "${:5432}", "", "camelCase"},
		{"malformed", "${database.port", "", "malformed"},
		{"nested default", "${database.port:${other}}", "", "malformed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			template := []byte("portalRules:\n- name: app.rule\n  matchPort: '" + tc.reference + "'\n")
			_, _, err := resolveSeedInputWithSchemas(template, []byte(tc.variables), nil, testVarsSchemas())
			require.ErrorContains(t, err, tc.want)
			if tc.name == "missing" {
				require.EqualError(t, err, tc.want)
			} else {
				require.Contains(t, err.Error(), `portalRule "app.rule" field "matchPort"`)
			}
			require.NotContains(t, err.Error(), "/portalRules/0/")
			require.NotContains(t, err.Error(), "secret-value")
		})
	}
}

func TestVarsTextDefaultsAndInterpolation(t *testing.T) {
	template := []byte(`portalRules:
- name: app.rule
  matchHost: "${text:https://localhost:8443/a.b}"
  matchPathPrefix: "${text:}/${text:second}"
  routeRedirectionPattern: "https://${database.host:localhost}:${database.port:8080}"
`)
	node, _, err := resolveSeedInputWithSchemas(template, nil, nil, testVarsSchemas())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.Equal(t, "https://localhost:8443/a.b", payload.PortalRules[0].MatchHost)
	require.Equal(t, "/second", payload.PortalRules[0].MatchPathPrefix)
	require.Equal(t, "https://localhost:8080", payload.PortalRules[0].RouteRedirectionPattern)
}

func TestVarsTypedDefaultKeepsNumericLookingString(t *testing.T) {
	node, _, err := resolveSeedInputWithSchemas([]byte("portalRules: [{name: app.rule, matchHost: '${text:00123}'}]"), nil, nil, testVarsSchemas())
	require.NoError(t, err)
	field := seedMappingValue(seedMappingValue(node, "portalRules").Content[0], "matchHost")
	require.Equal(t, "!!str", field.ShortTag())
	require.Equal(t, "00123", field.Value)
}

func TestVarsSchemaScalarValidation(t *testing.T) {
	for _, tc := range []struct {
		scalar skel.Scalar
		value  string
		valid  bool
	}{
		{skel.ScalarDuration, "2h", true}, {skel.ScalarDuration, "bad", false},
		{skel.ScalarTimestamp, "2026-09-14T10:30:00Z", true}, {skel.ScalarTimestamp, "bad", false},
		{skel.ScalarUuid, "550e8400-e29b-41d4-a716-446655440000", true}, {skel.ScalarUuid, "bad", false},
		{skel.ScalarInt, "5432", true}, {skel.ScalarInt, "'5432'", false},
		{skel.ScalarBool, "false", true}, {skel.ScalarBool, "'false'", false},
	} {
		var doc yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(tc.value), &doc))
		err := validateSeedScalar(doc.Content[0], tc.scalar)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestVarsLegacyJSONConfigInterpolation(t *testing.T) {
	node, _, err := resolveSeedInputWithSchemas([]byte(`appConfigs:
- name: app.DatabaseConfig
  value: '{"host":"${database.host:localhost}","port":${database.port:5432}}'
`), nil, nil, testVarsSchemas())
	require.NoError(t, err)
	var payload _SettingsYAMLPayload
	require.NoError(t, node.Decode(&payload))
	require.JSONEq(t, `{"host":"localhost","port":5432}`, payload.AppConfigs[0].Value)
}
