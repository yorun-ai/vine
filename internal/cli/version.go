package cli

import (
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
)

const (
	commandVersion = "version"

	flagVersionJSON = "json"

	cliName = "Vine CLI"
)

type _VersionInfo struct {
	Name string `json:"name"`
	buildinfo.DebugBuildInfo
	Dependencies _VersionDependencies `json:"dependencies"`
}

type _VersionDependencies struct {
	MinSkelcVersion string `json:"minSkelcVersion"`
}

func versionInfo() _VersionInfo {
	return _VersionInfo{
		Name:           cliName,
		DebugBuildInfo: buildinfo.MustDebugBuildInfo(),
		Dependencies: _VersionDependencies{
			MinSkelcVersion: skel.MinSkelcVersion(),
		},
	}
}

func (info _VersionInfo) TextString() string {
	return info.Name + "\n" +
		"  Version    " + info.Version + "\n" +
		"  Platform   " + info.Platform + "\n" +
		"  GoVersion  " + info.GoVersion + "\n" +
		"Dependencies:\n" +
		"  MinSkelcVersion  " + info.Dependencies.MinSkelcVersion
}

func (info _VersionInfo) JSONString() string {
	return vcode.MustMarshalJsonS(info)
}
