package vfile

import (
	"errors"
	"os"
	"strings"

	"go.yorun.ai/vine/util/vpre"
)

const defaultDirectoryMode = 0755

// EnsureCleanDirectory removes dirPath and recreates it as an empty directory.
func EnsureCleanDirectory(dirPath string) error {
	err := os.RemoveAll(dirPath)
	if err != nil {
		return err
	}
	return CreateDirectory(dirPath)
}

// CreateDirectory creates dirPath and missing parents using the package default mode.
func CreateDirectory(dirPath string) error {
	return os.MkdirAll(dirPath, defaultDirectoryMode)
}

// Exist reports whether path exists or cannot be confirmed absent.
func Exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

// ExistDir reports whether path identifies a directory.
func ExistDir(path string) bool {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return stat.IsDir()
}

// ExistsFile reports whether path identifies a non-directory filesystem entry.
func ExistsFile(path string) bool {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return !stat.IsDir()
}

// ExpandHomeDir expands a leading "~/" using the current user's home directory.
func ExpandHomeDir(dir string) string {
	if strings.HasPrefix(dir, "~/") {
		homeDir, err := os.UserHomeDir()
		vpre.CheckNilError(err, "get home directory failed")
		dir = strings.Replace(dir, "~", homeDir, 1)
	}
	return dir
}
