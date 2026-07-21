package testkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"go.yorun.ai/vine/core/conf"
	internalconf "go.yorun.ai/vine/internal/core/conf"
	"go.yorun.ai/vine/util/vcode"
	"gopkg.in/yaml.v3"
)

// ConfigOverride replaces one named application configuration value in a test runtime.
type ConfigOverride struct {
	// Name is the registered Skel name of the configuration.
	Name string
	// Value is serialized into the generated test seed configuration.
	Value any
}

// OverrideConfig creates an override whose name is derived from the registered config type C.
func OverrideConfig[C conf.Config](value C) ConfigOverride {
	kind := reflect.TypeOf(value)
	return ConfigOverride{
		Name:  internalconf.SkelNameByType(kind),
		Value: value,
	}
}

// OverrideConfigByName creates an override for an explicit Skel configuration name.
func OverrideConfigByName(name string, value any) ConfigOverride {
	return ConfigOverride{Name: name, Value: value}
}

func mergeSeedConfigOverrides(t testing.TB, seedPath string, overrides []ConfigOverride) (string, func(), error) {
	t.Helper()

	payload := map[string]any{}
	if seedPath != "" {
		content, err := os.ReadFile(seedPath)
		if err != nil {
			return "", nil, err
		}
		if err := yaml.Unmarshal(content, &payload); err != nil {
			return "", nil, err
		}
	}

	configs := seedAppConfigs(payload["appConfigs"])
	for _, override := range overrides {
		value, err := json.Marshal(override.Value)
		if err != nil {
			return "", nil, fmt.Errorf("marshal config override %s: %w", override.Name, err)
		}
		configs[override.Name] = string(value)
	}

	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)

	appConfigs := make([]map[string]any, 0, len(configs))
	for _, name := range names {
		appConfigs = append(appConfigs, map[string]any{
			"name":  name,
			"value": configs[name],
		})
	}
	payload["appConfigs"] = appConfigs

	path := filepath.Join(t.TempDir(), "vine-testkit-seed.yaml")
	if err := os.WriteFile(path, vcode.MustMarshalYaml(payload), 0o600); err != nil {
		return "", nil, err
	}
	return path, func() {}, nil
}

func seedAppConfigs(raw any) map[string]string {
	configs := map[string]string{}
	items, ok := raw.([]any)
	if !ok {
		return configs
	}
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, nameOK := itemMap["name"].(string)
		value, valueOK := itemMap["value"].(string)
		if nameOK && valueOK {
			configs[name] = value
		}
	}
	return configs
}
