package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameGetterReportsDefaultAndOverride(t *testing.T) {
	original := ldName
	t.Cleanup(func() {
		ldName = original
	})

	ldName = defaultName
	value, ok := Name()
	assert.Equal(t, defaultName, value)
	assert.False(t, ok)

	ldName = "user-service"
	value, ok = Name()
	assert.Equal(t, "user-service", value)
	assert.True(t, ok)
}

func TestVersionGetterReportsDefaultAndOverride(t *testing.T) {
	original := ldVersion
	t.Cleanup(func() {
		ldVersion = original
	})

	ldVersion = defaultVersion
	value, ok := Version()
	assert.Equal(t, defaultVersion, value)
	assert.False(t, ok)

	ldVersion = "1.2.3"
	value, ok = Version()
	assert.Equal(t, "1.2.3", value)
	assert.True(t, ok)
}

func TestBuildGettersReportDefaultAndOverride(t *testing.T) {
	originalCommit := ldGitCommit
	originalBuiltBy := ldBuiltBy
	originalBuiltTime := ldBuiltTime
	t.Cleanup(func() {
		ldGitCommit = originalCommit
		ldBuiltBy = originalBuiltBy
		ldBuiltTime = originalBuiltTime
	})

	ldGitCommit = defaultBuildText
	ldBuiltBy = defaultBuildText
	ldBuiltTime = defaultBuildText

	value, ok := GitCommit()
	assert.Equal(t, defaultBuildText, value)
	assert.False(t, ok)

	value, ok = BuiltBy()
	assert.Equal(t, defaultBuildText, value)
	assert.False(t, ok)

	value, ok = BuiltTime()
	assert.Equal(t, defaultBuildText, value)
	assert.False(t, ok)

	ldGitCommit = "abc123"
	ldBuiltBy = "ci"
	ldBuiltTime = "2026-04-17T00:00:00Z"

	value, ok = GitCommit()
	assert.Equal(t, "abc123", value)
	assert.True(t, ok)

	value, ok = BuiltBy()
	assert.Equal(t, "ci", value)
	assert.True(t, ok)

	value, ok = BuiltTime()
	assert.Equal(t, "2026-04-17T00:00:00Z", value)
	assert.True(t, ok)
}
