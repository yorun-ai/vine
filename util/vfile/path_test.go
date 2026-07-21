package vfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureCleanDirectoryAndCreateDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "nested", "dir")
	file := filepath.Join(dir, "payload.txt")

	assert.NoError(t, CreateDirectory(dir))
	assert.NoError(t, os.WriteFile(file, []byte("payload"), 0o644))
	assert.True(t, ExistDir(dir))
	assert.True(t, ExistsFile(file))

	assert.NoError(t, EnsureCleanDirectory(dir))
	assert.True(t, ExistDir(dir))
	assert.False(t, ExistsFile(file))
}

func TestExistHelpers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	file := filepath.Join(root, "file.txt")
	missing := filepath.Join(root, "missing")

	assert.NoError(t, os.MkdirAll(dir, 0o755))
	assert.NoError(t, os.WriteFile(file, []byte("content"), 0o644))

	assert.True(t, Exist(dir))
	assert.True(t, Exist(file))
	assert.False(t, Exist(missing))

	assert.True(t, ExistDir(dir))
	assert.False(t, ExistDir(file))
	assert.False(t, ExistDir(missing))

	assert.True(t, ExistsFile(file))
	assert.False(t, ExistsFile(dir))
	assert.False(t, ExistsFile(missing))
}

func TestExpandHomeDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	assert.Equal(t, filepath.Join(homeDir, "demo"), ExpandHomeDir("~/demo"))
	assert.Equal(t, "/tmp/demo", ExpandHomeDir("/tmp/demo"))
	assert.True(t, strings.HasPrefix(ExpandHomeDir("~/demo"), homeDir))
}
