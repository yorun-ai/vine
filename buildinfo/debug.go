package buildinfo

// This file exposes Go debug build information for command tools.

import (
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/vine/util/vpre"
)

// DevVersion is the fallback semantic version for an unversioned development build.
const DevVersion = "v0.0.0-dev"

// IsDevVersion reports whether version is the development fallback version.
func IsDevVersion(version string) bool {
	return version == DevVersion
}

var readBuildInfo = debug.ReadBuildInfo

// DebugBuildInfo contains module and Go toolchain metadata for diagnostic output.
type DebugBuildInfo struct {
	// Version is the normalized main module version.
	Version string `json:"version"`
	// Platform is the build target in GOOS/GOARCH form.
	Platform string `json:"platform"`
	// GoVersion is the Go toolchain version recorded in build information.
	GoVersion string `json:"goVersion"`
}

// MustDebugBuildInfo reads build information for the current executable or panics.
func MustDebugBuildInfo() DebugBuildInfo {
	readInfo, ok := readBuildInfo()
	vpre.Check(ok, "read Go build info failed")
	return DebugBuildInfo{
		Version:   moduleVersion(readInfo.Main.Version),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: readInfo.GoVersion,
	}
}

func moduleVersion(rawVersion string) string {
	version := rawVersion
	if version == "" || version == "(devel)" {
		return DevVersion
	}

	version = strings.TrimSuffix(version, "+dirty")
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	_, err := semver.NewVersion(version)
	vpre.CheckNilError(err, "parse module version %s failed", version)
	return version
}

// MustVineDependencyVersion returns the linked Vine module version.
// It returns DevVersion when the dependency has no valid semantic version.
func MustVineDependencyVersion() string {
	readInfo, ok := readBuildInfo()
	vpre.Check(ok, "read Go build info failed")

	for _, dep := range readInfo.Deps {
		if dep.Path != "go.yorun.ai/vine" {
			continue
		}
		if _, err := semver.NewVersion(dep.Version); err == nil {
			return dep.Version
		}
	}
	return DevVersion
}
