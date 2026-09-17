package buildinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoRuntimeInfoHelpers(t *testing.T) {
	assert.Equal(t, runtime.Version(), GoVersion())
	assert.Equal(t, runtime.Compiler, GoCompiler())
	assert.Equal(t, runtime.GOOS+"/"+runtime.GOARCH, GoPlatform())
}

func TestInspectReportsUnlinkedBuildValues(t *testing.T) {
	originalCommit, originalBuiltBy, originalBuiltTime := ldGitCommit, ldBuiltBy, ldBuiltTime
	t.Cleanup(func() {
		ldGitCommit, ldBuiltBy, ldBuiltTime = originalCommit, originalBuiltBy, originalBuiltTime
	})
	ldGitCommit, ldBuiltBy, ldBuiltTime = "", "", ""

	output := Inspect()
	assert.Contains(t, output, "Name = "+Name())
	assert.Contains(t, output, "Version = "+Version())
	assert.Contains(t, output, "GitCommit = "+unavailableText)
	assert.Contains(t, output, "BuiltBy = "+unavailableText)
	assert.Contains(t, output, "BuiltTime = "+unavailableText)
}
