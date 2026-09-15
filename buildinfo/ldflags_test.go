package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkerInjectedGettersReportDefaultAndOverride(t *testing.T) {
	originalName, originalVersion := ldName, ldVersion
	originalCommit, originalBuiltBy, originalBuiltTime := ldGitCommit, ldBuiltBy, ldBuiltTime
	t.Cleanup(func() {
		ldName, ldVersion = originalName, originalVersion
		ldGitCommit, ldBuiltBy, ldBuiltTime = originalCommit, originalBuiltBy, originalBuiltTime
	})

	for _, tt := range []struct {
		name     string
		target   *string
		get      func() (string, bool)
		fallback string
		override string
	}{
		{
			name:     "Name",
			target:   &ldName,
			get:      Name,
			fallback: defaultName,
			override: "user-service",
		},
		{
			name:     "Version",
			target:   &ldVersion,
			get:      Version,
			fallback: defaultVersion,
			override: "1.2.3",
		},
		{
			name:     "GitCommit",
			target:   &ldGitCommit,
			get:      GitCommit,
			fallback: defaultBuildText,
			override: "abc123",
		},
		{
			name:     "BuiltBy",
			target:   &ldBuiltBy,
			get:      BuiltBy,
			fallback: defaultBuildText,
			override: "ci",
		},
		{
			name:     "BuiltTime",
			target:   &ldBuiltTime,
			get:      BuiltTime,
			fallback: defaultBuildText,
			override: "2026-04-17T00:00:00Z",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			*tt.target = tt.fallback
			value, ok := tt.get()
			assert.Equal(t, tt.fallback, value)
			assert.False(t, ok)

			*tt.target = tt.override
			value, ok = tt.get()
			assert.Equal(t, tt.override, value)
			assert.True(t, ok)
		})
	}
}
