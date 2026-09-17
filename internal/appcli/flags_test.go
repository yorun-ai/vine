package appcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
