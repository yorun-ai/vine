package appcli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	ucli "github.com/urfave/cli/v3"
	skeldesc "go.yorun.ai/skel/descriptor"
)

func variableFlagsForTest(target *[]string, ignored []string, renamed map[string]string, paths map[string]string) []ucli.Flag {
	names := NewFlagNames(ignored, renamed)
	seed := names.StringSlice("hub-seed-var", "VINE_TEST_SEED_VAR", target, "seed variable")
	list := append([]ucli.Flag{seed}, names.VariableFlags(paths, target, seed)...)
	names.Validate()
	return list
}

func TestVariableFlagsPreserveCommandLineOrder(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "text=env-general")
	t.Setenv("DB_HOST", "env-host")
	t.Setenv("DB_PORT", "5432")
	var assignments []string
	flags := variableFlagsForTest(&assignments, nil, nil, map[string]string{
		"database.host": "db-host", "database.port": "db-port", "origins": "origins",
	})
	_, err := parseArgs([]string{"app",
		"--hub-seed-var", "database.host=general-first",
		"--db-host", "named",
		"--origins", `["https://a.com","https://b.com"]`,
		"--db-host=https://host/?a=b,c",
		"--hub-seed-var=database.host=general-last",
	}, flags...)
	require.NoError(t, err)
	require.Equal(t, []string{
		"database.port=5432",
		"database.host=general-first", "database.host=named",
		`origins=["https://a.com","https://b.com"]`, "database.host=https://host/?a=b,c",
		"database.host=general-last",
	}, assignments)
}

func TestVariableFlagEnvironmentPrecedesGeneralCommandLine(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "text=unused")
	t.Setenv("DB_HOST", "env-host")
	var assignments []string
	_, err := parseArgs([]string{"app", "--hub-seed-var", "database.host=cli"},
		variableFlagsForTest(&assignments, nil, nil, map[string]string{"database.host": "db-host"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"database.host=env-host", "database.host=cli"}, assignments)
}

func TestVariableFlagEnvironmentOnlyAndEmptyValue(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "database.host=general-env")
	t.Setenv("DB_HOST", "host-env")
	t.Setenv("VINE_DB_HOST", "unused-prefixed-env")
	t.Setenv("DB_PORT", "5432")
	var assignments []string
	_, err := parseArgs([]string{"app"}, variableFlagsForTest(&assignments, nil, nil,
		map[string]string{"database.host": "db-host", "database.port": "db-port"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"database.host=general-env", "database.host=host-env", "database.port=5432"}, assignments)

	assignments = nil
	_, err = parseArgs([]string{"app", "--db-host="}, variableFlagsForTest(&assignments,
		[]string{"hub-seed-var"}, nil, map[string]string{"database.host": "db-host"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"database.host="}, assignments)
}

func TestVariableFlagsWorkWithIgnoredAndRenamedSeedFlag(t *testing.T) {
	for _, ignored := range []bool{false, true} {
		t.Run(map[bool]string{false: "renamed", true: "ignored"}[ignored], func(t *testing.T) {
			t.Setenv("VINE_TEST_SEED_VAR", "text=old-env")
			t.Setenv("DB_HOST", "env")
			var assignments []string
			var ignore []string
			var rename map[string]string
			args := []string{"app", "--db-host", "host"}
			if ignored {
				ignore = []string{"hub-seed-var"}
				args = append(args, "--hub-seed-var", "database.host=ignored")
			} else {
				rename = map[string]string{"hub-seed-var": "var"}
				args = append(args, "--var", "database.port=5432")
			}
			_, err := parseArgs(args, variableFlagsForTest(&assignments, ignore, rename,
				map[string]string{"database.host": "db-host"})...)
			require.NoError(t, err)
			if ignored {
				require.Equal(t, []string{"database.host=host"}, assignments)
			} else {
				require.Equal(t, []string{"database.host=host", "database.port=5432"}, assignments)
			}
		})
	}
}

func TestVariableFlagsRejectInvalidDeclarations(t *testing.T) {
	for _, paths := range []map[string]string{
		{"": "db"}, {"a..b": "db"}, {"a_b": "db"}, {"a.0": "db"},
		{"a": "Bad"}, {"a": ""}, {"a": "log-level"}, {"a": "log-rule"},
		{"a": "hub-seed-var"}, {"a": "shared", "b": "shared"},
	} {
		require.Panics(t, func() { variableFlagsForTest(new([]string), nil, nil, paths) })
	}
	require.Panics(t, func() {
		variableFlagsForTest(new([]string), nil, map[string]string{"hub-seed-var": "var"}, map[string]string{"a": "var"})
	})
}

func TestVariableFlagsRejectEnvironmentCollisions(t *testing.T) {
	for _, name := range []string{"vine-test-seed-var", "vine-log-level", "vine-log-rules"} {
		t.Run(name, func(t *testing.T) {
			require.Panics(t, func() {
				variableFlagsForTest(new([]string), nil, nil, map[string]string{"database.host": name})
			})
		})
	}
	// Renaming no longer reserves the automatically prefixed business name.
	require.NotPanics(t, func() {
		variableFlagsForTest(new([]string), nil, map[string]string{"hub-seed-var": "var"},
			map[string]string{"database.host": "vine-var"})
	})
	// An ignored flag still parses its environment input and reserves that name.
	require.Panics(t, func() {
		variableFlagsForTest(new([]string), []string{"hub-seed-var"}, nil,
			map[string]string{"database.host": "vine-test-seed-var"})
	})
	// Renaming releases the original environment name.
	require.NotPanics(t, func() {
		variableFlagsForTest(new([]string), nil, map[string]string{"hub-seed-var": "var"},
			map[string]string{"database.host": "vine-test-seed-var"})
	})
}

func TestVariableFlagPreservesEmptyEnvironmentValue(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "database.host=file-replacement")
	t.Setenv("DB_HOST", "")
	var assignments []string
	_, err := parseArgs([]string{"app"}, variableFlagsForTest(&assignments, nil, nil,
		map[string]string{"database.host": "db-host"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"database.host=file-replacement", "database.host="}, assignments)

	assignments = nil
	_, err = parseArgs([]string{"app", "--db-host", "cli"}, variableFlagsForTest(&assignments,
		[]string{"hub-seed-var"}, nil, map[string]string{"database.host": "db-host"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"database.host=cli"}, assignments)
}

func variableBoolFlagsForTest(target *[]string, ignored []string, renamed map[string]string, paths map[string]string) []ucli.Flag {
	names := NewFlagNames(ignored, renamed)
	seed := names.StringSlice("hub-seed-var", "VINE_TEST_SEED_VAR", target, "seed variable")
	list := append([]ucli.Flag{seed}, names.variableFlags(paths, target, seed, variableFlagTestDescriptors())...)
	names.Validate()
	return list
}

func unsetVariableBoolEnv(t *testing.T, name string) {
	t.Helper()
	t.Setenv(name, "")
	require.NoError(t, os.Unsetenv(name))
}

func TestBooleanVariableFlagsPresenceAndRepeats(t *testing.T) {
	unsetVariableBoolEnv(t, "ENABLED")
	t.Setenv("VINE_TEST_SEED_VAR", "")
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "absent"},
		{name: "bare", args: []string{"--enabled"}, want: []string{"enabled=true"}},
		{name: "false", args: []string{"--enabled=false"}, want: []string{"enabled=false"}},
		{name: "repeat", args: []string{"--enabled=false", "--enabled", "--enabled=true"}, want: []string{"enabled=false", "enabled=true", "enabled=true"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var assignments []string
			_, err := parseArgs(append([]string{"app"}, test.args...), variableBoolFlagsForTest(&assignments,
				nil, nil, map[string]string{"enabled": "enabled"})...)
			require.NoError(t, err)
			require.Equal(t, test.want, assignments)
		})
	}
}

func TestBooleanVariableEnvironment(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "")
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ENABLED", value)
			var assignments []string
			_, err := parseArgs([]string{"app"}, variableBoolFlagsForTest(&assignments,
				nil, nil, map[string]string{"enabled": "enabled"})...)
			require.NoError(t, err)
			require.Equal(t, []string{"enabled=" + value}, assignments)
		})
	}
	for _, value := range []string{"", "wrong", "null"} {
		t.Run("invalid-"+value, func(t *testing.T) {
			t.Setenv("ENABLED", value)
			var assignments []string
			_, err := parseArgs([]string{"app"}, variableBoolFlagsForTest(&assignments,
				nil, nil, map[string]string{"enabled": "enabled"})...)
			require.ErrorContains(t, err, "expected boolean value")
			require.Empty(t, assignments)
		})
	}
}

func TestBooleanVariableOrderAndEnvironmentOverride(t *testing.T) {
	t.Setenv("ENABLED", "invalid-but-overridden")
	t.Setenv("VINE_TEST_SEED_VAR", "")
	unsetVariableBoolEnv(t, "TEXT")
	var assignments []string
	_, err := parseArgs([]string{"app", "--enabled", "--text", "next", "--hub-seed-var", "enabled=false", "--enabled=false"},
		variableBoolFlagsForTest(&assignments, nil, nil, map[string]string{"enabled": "enabled", "text": "text"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"enabled=true", "text=next", "enabled=false", "enabled=false"}, assignments)

	assignments = nil
	t.Setenv("ENABLED", "true")
	_, err = parseArgs([]string{"app", "--hub-seed-var", "enabled=false"},
		variableBoolFlagsForTest(&assignments, nil, nil, map[string]string{"enabled": "enabled"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"enabled=true", "enabled=false"}, assignments)
}

func TestBooleanVariableWorksWithIgnoredOrRenamedSeedFlag(t *testing.T) {
	unsetVariableBoolEnv(t, "ENABLED")
	t.Setenv("VINE_TEST_SEED_VAR", "")
	var assignments []string
	_, err := parseArgs([]string{"app", "--enabled"}, variableBoolFlagsForTest(&assignments,
		[]string{"hub-seed-var"}, nil, map[string]string{"enabled": "enabled"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"enabled=true"}, assignments)

	assignments = nil
	t.Setenv("VAR", "")
	_, err = parseArgs([]string{"app", "--enabled", "--var", "enabled=false"}, variableBoolFlagsForTest(&assignments,
		nil, map[string]string{"hub-seed-var": "var"}, map[string]string{"enabled": "enabled"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"enabled=true", "enabled=false"}, assignments)
}

func TestNullableBooleanVariableAcceptsExplicitNull(t *testing.T) {
	t.Setenv("VINE_TEST_SEED_VAR", "")
	t.Setenv("OPTIONAL", "null")
	var assignments []string
	_, err := parseArgs([]string{"app"}, variableBoolFlagsForTest(&assignments,
		nil, nil, map[string]string{"optional": "optional"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"optional=null"}, assignments)
	assignments = nil
	_, err = parseArgs([]string{"app", "--optional", "--optional=null"}, variableBoolFlagsForTest(&assignments,
		nil, nil, map[string]string{"optional": "optional"})...)
	require.NoError(t, err)
	require.Equal(t, []string{"optional=true", "optional=null"}, assignments)
}

func TestVariableFlagBooleanSelectionAndFallback(t *testing.T) {
	var assignments []string
	flags := variableBoolFlagsForTest(&assignments, nil, nil, map[string]string{
		"feature.enabled": "feature-enabled", "text": "text", "unknown": "unknown",
	})
	for _, flag := range flags[1:] {
		takesValue := flag.(interface{ TakesValue() bool }).TakesValue()
		require.Equal(t, flag.Names()[0] != "feature-enabled", takesValue)
	}
	// Without app.Vars, even a bool-looking path still requires a YAML value.
	_, err := parseArgs([]string{"app", "--enabled"}, variableFlagsForTest(&assignments,
		nil, nil, map[string]string{"enabled": "enabled"})...)
	require.Error(t, err)
	_, err = parseArgs([]string{"app", "--unknown"}, variableBoolFlagsForTest(&assignments,
		nil, nil, map[string]string{"unknown": "unknown"})...)
	require.Error(t, err)
	_, err = parseArgs([]string{"app", "--enabled=wrong"}, variableBoolFlagsForTest(&assignments,
		nil, nil, map[string]string{"enabled": "enabled"})...)
	require.ErrorContains(t, err, "expected boolean value")
}

func variableFlagTestDescriptors() []*skeldesc.Domain {
	boolean := new(skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarBoolean,
	})
	text := new(skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarString,
	})
	return []*skeldesc.Domain{
		{
			Data: []*skeldesc.Data{
				{
					SkelName: "app.Vars",
					Members: []*skeldesc.Member{
						{
							Name: "enabled",
							Type: boolean,
						},
						{
							Name: "optional",
							Type: new(skeldesc.Type{
								Kind:     skeldesc.TypeKindScalar,
								Scalar:   skeldesc.ScalarBoolean,
								Nullable: true,
							}),
						},
						{
							Name: "feature",
							Type: new(skeldesc.Type{
								Kind:     skeldesc.TypeKindData,
								SkelName: "app.Feature",
							}),
						},
						{
							Name: "config",
							Type: new(skeldesc.Type{
								Kind:     skeldesc.TypeKindConfig,
								SkelName: "app.Config",
							}),
						},
						{
							Name: "toggles",
							Type: new(skeldesc.Type{
								Kind:  skeldesc.TypeKindMap,
								Key:   text,
								Value: boolean,
							}),
						},
						{
							Name: "groups",
							Type: new(skeldesc.Type{
								Kind:    skeldesc.TypeKindList,
								Element: boolean,
							}),
						},
						{
							Name: "text",
							Type: text,
						},
						{
							Name: "missingType",
							Type: new(skeldesc.Type{
								Kind:     skeldesc.TypeKindData,
								SkelName: "app.Missing",
							}),
						},
					},
				},
				{
					SkelName: "app.Feature",
					Members: []*skeldesc.Member{
						{
							Name: "enabled",
							Type: boolean,
						},
					},
				},
			},
			Configs: []*skeldesc.Config{
				{
					SkelName: "app.Config",
					Members: []*skeldesc.Member{
						{
							Name: "enabled",
							Type: boolean,
						},
					},
					Lifecycle: skeldesc.ConfigLifecycleEternal,
				},
			},
			Generated: &skeldesc.GeneratedInfo{
				CompilerVersion: "v99.0.0",
			},
		},
	}
}

func TestVariableFlagDescriptorResolvesBooleanPaths(t *testing.T) {
	descriptor := newVariableFlagDescriptor(variableFlagTestDescriptors())
	for _, path := range []string{"enabled", "feature.enabled", "config.enabled", "toggles.anyKey", "optional"} {
		t.Run(path, func(t *testing.T) {
			kind := descriptor.variableType(path)
			require.NotNil(t, kind)
			require.Equal(t, skeldesc.TypeKindScalar, kind.Kind)
			require.Equal(t, skeldesc.ScalarBoolean, kind.Scalar)
		})
	}
	require.True(t, descriptor.variableType("optional").Nullable)
	require.Equal(t, skeldesc.ScalarString, descriptor.variableType("text").Scalar)
	for _, path := range []string{"unknown", "feature.unknown", "enabled.child", "groups.item", "missingType.enabled"} {
		require.Nil(t, descriptor.variableType(path))
	}
	require.Nil(t, newVariableFlagDescriptor(nil).variableType("enabled"))
}
