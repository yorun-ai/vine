package vfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTouchAndMustTouch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "touch.txt")

	assert.NoError(t, Touch(path))
	assert.True(t, ExistsFile(path))

	mustPath := filepath.Join(root, "must-touch.txt")
	assert.NotPanics(t, func() {
		MustTouch(mustPath)
	})
	assert.True(t, ExistsFile(mustPath))
}

func TestWriteString(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "deep", "nested", "file.txt")

	assert.NoError(t, WriteString(path, "written content"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "written content", string(content))
}
