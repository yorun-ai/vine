package appcli

import (
	"testing"

	"github.com/stretchr/testify/require"
	ucli "github.com/urfave/cli/v3"
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
	require.Panics(t, func() {
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
