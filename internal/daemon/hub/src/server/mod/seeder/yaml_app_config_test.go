package seeder

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAppConfigStructuredYAML(t *testing.T) {
	const input = `appConfigs:
  - name: demo.Config
    override: true
    value:
      displayName: Demo
      statuses: {EAST: ACTIVE, WEST: LOCKED}
      numericKeys: {1: ACTIVE, 9007199254740993: LARGE}
      enabled: true
      count: 42
      optional: null
      date: 2026-09-13
      timestamp: 2026-09-13T10:15:30.123456789+08:00
      items: [one, two]
      message: |
        first
        second
`
	var settings _SettingsYAMLPayload
	require.NoError(t, yaml.Unmarshal([]byte(input), &settings))
	require.Len(t, settings.AppConfigs, 1)
	item := settings.AppConfigs[0]
	require.True(t, item.Override)
	require.Equal(t, "demo.Config", item.Name)
	var value map[string]any
	require.NoError(t, json.Unmarshal([]byte(item.Value), &value))
	require.Equal(t, "Demo", value["displayName"])
	require.Equal(t, map[string]any{"EAST": "ACTIVE", "WEST": "LOCKED"}, value["statuses"])
	require.Equal(t, map[string]any{"1": "ACTIVE", "9007199254740993": "LARGE"}, value["numericKeys"])
	require.Equal(t, true, value["enabled"])
	require.Equal(t, float64(42), value["count"])
	require.Nil(t, value["optional"])
	require.Equal(t, "2026-09-13", value["date"])
	require.Equal(t, "2026-09-13T10:15:30.123456789+08:00", value["timestamp"])
	require.Equal(t, []any{"one", "two"}, value["items"])
	require.Equal(t, "first\nsecond\n", value["message"])
	require.Equal(t, item.Value, item.ToCoreAppConfig().Value)
	var again _SettingsYAMLPayload
	require.NoError(t, yaml.Unmarshal([]byte(input), &again))
	require.Equal(t, item.Value, again.AppConfigs[0].Value)
}

func TestAppConfigLegacyAndScalarYAML(t *testing.T) {
	for _, test := range []struct{ input, expected string }{
		{`'{"enabled":"yes","maxBatchSize":"many"}'`, `{"enabled":"yes","maxBatchSize":"many"}`},
		{`'"legacy scalar"'`, `"legacy scalar"`},
		{`'invalid JSON fixture'`, `invalid JSON fixture`},
		{`true`, `true`},
		{`42`, `42`},
		{`null`, `null`},
		{`[ACTIVE, LOCKED]`, `["ACTIVE","LOCKED"]`},
		{`{}`, `{}`},
		{`{"<<": "literal"}`, `{"<<":"literal"}`},
	} {
		t.Run(test.input, func(t *testing.T) {
			var item _AppConfig
			require.NoError(t, yaml.Unmarshal([]byte("name: demo.Config\nvalue: "+test.input), &item))
			require.Equal(t, test.expected, item.Value)
		})
	}
}

func TestAppConfigYAMLRejectsAnchorsAndAliases(t *testing.T) {
	for _, input := range []string{
		"name: demo.Config\nvalue: &value {enabled: true}",
		"name: demo.Config\nvalue: &value '{\"enabled\":true}'",
		"name: demo.Config\nvalue: {a: &a 1, b: *a}",
		"name: demo.Config\nvalue: &loop {self: *loop}",
		"name: &name demo.Config\nvalue: *name",
	} {
		t.Run(input, func(t *testing.T) {
			var item _AppConfig
			err := yaml.Unmarshal([]byte(input), &item)
			require.ErrorContains(t, err, "YAML anchors and aliases are not supported")
		})
	}
}

func TestAppConfigYAMLRejectsValuesWithoutJSONRepresentation(t *testing.T) {
	for _, value := range []string{
		`{a: 1, a: 2}`, `{a: .inf}`, `{a: .nan}`, `{a: !custom text}`,
		`{1: first, "1": second}`, `{<<: {enabled: true}}`, `{nested: {<<: {enabled: true}}}`,
		`{? [a, b]: value}`, `{null: value}`,
	} {
		t.Run(value, func(t *testing.T) {
			var item _AppConfig
			err := yaml.Unmarshal([]byte("name: demo.Config\nvalue: "+value), &item)
			require.Error(t, err)
			require.Contains(t, err.Error(), "demo.Config")
		})
	}
}
