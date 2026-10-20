package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckLinkerInjectedIdentity(t *testing.T) {
	originalName, originalVersion := ldName, ldVersion
	t.Cleanup(func() {
		ldName, ldVersion = originalName, originalVersion
	})

	ldVersion = defaultVersion
	for _, name := range []string{
		defaultName,
		"user.service",
		"user-service",
		"app2",
		"app.base.user",
		"demo.worker-2",
		"team-a-1.app-b2-34",
	} {
		ldName = name
		assert.NotPanics(t, checkLinkerInjectedIdentity)
	}
	for _, name := range []string{
		"",
		"UserService",
		"user_service",
		"app..base",
		"-app",
		"app-",
		"app--base",
		"app.-base",
		"app-.base",
		"2app",
		".app",
		"app.",
	} {
		ldName = name
		assert.Panics(t, checkLinkerInjectedIdentity, "name %q must be rejected", name)
	}

	ldName = defaultName
	for _, version := range []string{defaultVersion, "1.2.3", "v1.2.3", "0.0.0-dev.abc123", "v1.2.4-4-gabc123", "1.2.3+build.5"} {
		ldVersion = version
		assert.NotPanics(t, checkLinkerInjectedIdentity)
	}
	for _, version := range []string{"", "latest", "1", "1.2", "1.2.3.4", "V1.2.3"} {
		ldVersion = version
		assert.Panics(t, checkLinkerInjectedIdentity, "version %q must be rejected", version)
	}
}
