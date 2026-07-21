package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/buildinfo"
)

func TestBuiltHelpersProxyTopLevelRuntime(t *testing.T) {
	gitCommit, _ := buildinfo.GitCommit()
	builtBy, _ := buildinfo.BuiltBy()
	builtTime, _ := buildinfo.BuiltTime()

	assert.Equal(t, gitCommit, GitCommit())
	assert.Equal(t, builtBy, BuiltBy())
	assert.Equal(t, builtTime, BuiltTime())
}
