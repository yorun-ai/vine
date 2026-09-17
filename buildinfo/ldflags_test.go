package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkerInjectedTextGettersReportEmptyDefaultAndOverride(t *testing.T) {
	originalCommit, originalBuiltBy, originalBuiltTime := ldGitCommit, ldBuiltBy, ldBuiltTime
	t.Cleanup(func() {
		ldGitCommit, ldBuiltBy, ldBuiltTime = originalCommit, originalBuiltBy, originalBuiltTime
	})

	for _, tt := range []struct {
		name     string
		target   *string
		get      func() string
		override string
	}{
		{
			name:     "GitCommit",
			target:   &ldGitCommit,
			get:      GitCommit,
			override: "abc123",
		},
		{
			name:     "BuiltBy",
			target:   &ldBuiltBy,
			get:      BuiltBy,
			override: "ci",
		},
		{
			name:     "BuiltTime",
			target:   &ldBuiltTime,
			get:      BuiltTime,
			override: "2026-04-17T00:00:00Z",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			*tt.target = ""
			assert.Empty(t, tt.get())

			*tt.target = tt.override
			assert.Equal(t, tt.override, tt.get())
		})
	}
}

func TestLinkerInjectedNameAndVersionReportDefaultAndOverride(t *testing.T) {
	originalName, originalVersion := ldName, ldVersion
	t.Cleanup(func() {
		ldName, ldVersion = originalName, originalVersion
	})

	ldName, ldVersion = defaultName, defaultVersion
	assert.Equal(t, defaultName, Name())
	assert.Equal(t, defaultVersion, Version())

	ldName, ldVersion = "user.service", "v1.2.3"
	assert.Equal(t, "user.service", Name())
	assert.Equal(t, "v1.2.3", Version())
}

func TestValidLinkerValues(t *testing.T) {
	for _, name := range []string{"vined", "user.service", "user-service", "demo.worker-2"} {
		assert.True(t, IsValidName(name), "name %q must be accepted", name)
	}
	for _, name := range []string{"", "User", "2app", "app-", "user_service"} {
		assert.False(t, IsValidName(name), "name %q must be rejected", name)
	}

	for _, version := range []string{"0.0.0", "1.2.3", "v1.2.3", "0.0.0-dev.abc123", "1.2.3+build.5"} {
		assert.True(t, IsValidVersion(version), "version %q must be accepted", version)
	}
	for _, version := range []string{"", "latest", "1", "1.2", "V1.2.3"} {
		assert.False(t, IsValidVersion(version), "version %q must be rejected", version)
	}
}

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
