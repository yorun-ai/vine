package logger

import (
	"log/slog"
	"path"
	"path/filepath"
	"strings"
)

func trimSourceFile(source *slog.Source) string {
	cleanFile := filepath.ToSlash(filepath.Clean(source.File))
	if source.Function == "" && strings.HasPrefix(cleanFile, "STDLOG") {
		return cleanFile
	}

	return shortCallerFile(cleanFile)
}

func shortCallerFile(file string) string {
	dir := path.Base(path.Dir(file))
	base := path.Base(file)
	if dir == "." || dir == "/" {
		return base
	}
	return dir + "/" + base
}

func trimFunctionPackage(function string) string {
	if function == "" {
		return ""
	}

	slashIndex := strings.LastIndex(function, "/")
	if slashIndex < 0 {
		return ""
	}

	dotIndex := strings.Index(function[slashIndex+1:], ".")
	if dotIndex < 0 {
		return ""
	}

	return function[:slashIndex+1+dotIndex]
}
