package vfile

import (
	"os"
	"path/filepath"

	"go.yorun.ai/vine/util/vpre"
)

const defaultFileMode = 0644

// Touch creates path when absent without truncating an existing file.
func Touch(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, defaultFileMode)
	if err != nil {
		return err
	}
	return file.Close()
}

// MustTouch is like Touch but panics on failure.
func MustTouch(path string) {
	err := Touch(path)
	vpre.CheckNilError(err, "touch %s failed", path)
}

// WriteString creates parent directories and writes content to path.
func WriteString(path string, content string) error {
	err := CreateDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), defaultFileMode)
}
