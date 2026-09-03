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

	pathPrefix, packageFunction, ok := strings.CutLast(function, "/")
	if !ok {
		return ""
	}

	dotIndex := strings.Index(packageFunction, ".")
	if dotIndex < 0 {
		return ""
	}

	return pathPrefix + "/" + packageFunction[:dotIndex]
}
