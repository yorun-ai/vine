package appcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two flags sharing a registered name leave one of them unreachable, so the
// declaration is refused instead of dropping it.
func TestFlagNamesRejectsCollidingNames(t *testing.T) {
	names := NewFlagNames(nil, map[string]string{"hub-first": "shared", "hub-second": "shared"})

	assert.Panics(t, func() {
		names.String("hub-first", "VINE_FIRST", new(string), "first")
		names.String("hub-second", "VINE_SECOND", new(string), "second")
	})
}

// The command line owns the logging flags, so a declared or renamed name cannot
// take them over.
func TestFlagNamesRejectsTheCommandLineNames(t *testing.T) {
	for _, name := range []string{flagLogLevel, flagLogRule} {
		t.Run(name, func(t *testing.T) {
			names := NewFlagNames(nil, map[string]string{"hub-endpoint": name})

			assert.Panics(t, func() {
				names.String("hub-endpoint", "VINE_HUB_ENDPOINT", new(string), "endpoint")
			})
		})
	}
}

// A renamed flag carries the environment variable derived from its new name, so
// only a name that derives a variable a shell can set is accepted.
func TestFlagNamesRejectsMalformedNames(t *testing.T) {
	valid := NewFlagNames(nil, map[string]string{"hub-endpoint": "admin-listen-2"})
	assert.NotPanics(t, func() {
		valid.String("hub-endpoint", "VINE_HUB_ENDPOINT", new(string), "endpoint")
	})

	for _, name := range []string{"Admin-Listen", "admin.listen", "admin listen", "-admin", "admin-", "2admin", "admin_key"} {
		t.Run(name, func(t *testing.T) {
			names := NewFlagNames(nil, map[string]string{"hub-endpoint": name})

			assert.Panics(t, func() {
				names.String("hub-endpoint", "VINE_HUB_ENDPOINT", new(string), "endpoint")
			})
		})
	}
}

func TestFlagNamesRejectsCollidingEnvironmentVariables(t *testing.T) {
	for _, ignored := range []bool{false, true} {
		var ignore []string
		if ignored {
			ignore = []string{"first"}
		}
		names := NewFlagNames(ignore, nil)
		names.String("first", "SHARED", new(string), "first")
		assert.PanicsWithError(t, `environment variable "SHARED" is registered for both flags "first" and "second"`, func() {
			names.Bool("second", "SHARED", new(bool), "second")
		})
	}
}

func TestRenamedFlagEnvironmentIsRegistered(t *testing.T) {
	names := NewFlagNames(nil, map[string]string{"original": "renamed"})
	names.String("original", "OLD_ENV", new(string), "original")
	assert.NotPanics(t, func() { names.String("other", "OLD_ENV", new(string), "other") })
	assert.PanicsWithError(t, `environment variable "RENAMED" is registered for both flags "renamed" and "third"`, func() {
		names.StringSlice("third", "RENAMED", new([]string), "third")
	})
}

func TestFlagNamesReservesLoggingEnvironmentVariables(t *testing.T) {
	for _, env := range []string{envLogLevel, envLogRules} {
		names := NewFlagNames(nil, nil)
		assert.Panics(t, func() { names.String("other", env, new(string), "other") })
	}
}

func TestRenamedFlagsReadOnlyUnprefixedEnvironment(t *testing.T) {
	t.Setenv("OLD_TEXT", "old")
	t.Setenv("VINE_TEXT", "prefixed")
	t.Setenv("TEXT", "business")
	t.Setenv("OLD_ENABLED", "false")
	t.Setenv("VINE_ENABLED", "false")
	t.Setenv("ENABLED", "true")
	t.Setenv("OLD_VARS", "text=old")
	t.Setenv("VINE_VARS", "text=prefixed")
	t.Setenv("VARS", "text=business")
	names := NewFlagNames(nil, map[string]string{
		"original-text": "text", "original-enabled": "enabled", "original-vars": "vars",
	})
	var value string
	var enabled bool
	var assignments []string
	_, err := parseArgs([]string{"app"},
		names.String("original-text", "OLD_TEXT", &value, "text"),
		names.Bool("original-enabled", "OLD_ENABLED", &enabled, "enabled"),
		names.StringSlice("original-vars", "OLD_VARS", &assignments, "vars"))
	require.NoError(t, err)
	require.Equal(t, "business", value)
	require.True(t, enabled)
	require.Equal(t, []string{"text=business"}, assignments)
}

func TestRenamedFlagRejectsLoggingEnvironmentCollision(t *testing.T) {
	names := NewFlagNames(nil, map[string]string{"original": "vine-log-level"})
	assert.PanicsWithError(t, `environment variable "VINE_LOG_LEVEL" is registered for both flags "log-level" and "vine-log-level"`, func() {
		names.String("original", "OLD_ENV", new(string), "original")
	})
}
