package runtime

import (
	goruntime "runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGolangHelpersExposeGoRuntimeInfo(t *testing.T) {
	assert.Equal(t, goruntime.Version(), GoVersion())
	assert.Equal(t, goruntime.Compiler, GoCompiler())
	assert.Equal(t, goruntime.GOOS+"/"+goruntime.GOARCH, GoPlatform())
}
