package buildinfo

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/vine/util/vpre"
)

// linkerNameSegment matches one dot-separated segment of an executable name: the
// shape of an application name segment with dashes allowed between letters and
// digits, so a segment never starts or ends with a dash.
const linkerNameSegment = `[a-z][a-z0-9]*(?:-[a-z0-9]+)*`

// linkerNamePattern matches the executable name a build may link: an application
// name whose segments carry dashes, such as user.service, user-service, or
// demo.worker-2.
var linkerNamePattern = regexp.MustCompile(`^` + linkerNameSegment + `(?:\.` + linkerNameSegment + `)*$`)

// IsValidName reports whether name can be linked as the executable name.
func IsValidName(name string) bool {
	return linkerNamePattern.MatchString(name)
}

// IsValidVersion reports whether version can be linked as the application
// version: a full semantic version, with the Go module form's leading "v"
// accepted.
func IsValidVersion(version string) bool {
	_, err := semver.StrictNewVersion(strings.TrimPrefix(version, "v"))
	return err == nil
}

// init checks the identity a build linked. A name or version the framework
// cannot use panics while the process starts instead of surfacing later from a
// getter or reaching application metadata.
func init() {
	checkLinkerInjectedIdentity()
}

// checkLinkerInjectedIdentity validates the linker-injected name and version.
func checkLinkerInjectedIdentity() {
	vpre.Check(IsValidName(ldName), "invalid ldName: %q, lowercase letters, digits, dots and dashes expected", ldName)
	vpre.Check(IsValidVersion(ldVersion), "invalid ldVersion: %q, semantic version expected", ldVersion)
}

// Name returns the linker-injected executable name. A build that links an
// unusable name fails when the process starts.
func Name() string {
	return ldName
}

// Version returns the linker-injected version. It is a full semantic version,
// and the Go module form with a leading "v" is accepted. A build that links an
// unusable version fails when the process starts.
func Version() string {
	return ldVersion
}

// GitCommit returns the linker-injected commit, or an empty string when the
// build linked none.
func GitCommit() string {
	return ldGitCommit
}

// BuiltBy returns the linker-injected builder, or an empty string when the
// build linked none.
func BuiltBy() string {
	return ldBuiltBy
}

// BuiltTime returns the linker-injected build time, or an empty string when the
// build linked none.
func BuiltTime() string {
	return ldBuiltTime
}

// GoVersion returns the Go toolchain version used to build the executable.
func GoVersion() string {
	return runtime.Version()
}

// GoCompiler returns the compiler used to build the executable.
func GoCompiler() string {
	return runtime.Compiler
}

// GoPlatform returns the target operating system and architecture of the executable.
func GoPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

// unavailableText is how Inspect reports a build value the build did not link.
const unavailableText = "NotAvailable"

// Inspect returns a human-readable summary of the linked executable identity
// and the toolchain that produced it.
func Inspect() string {
	desc := "AppInfo\n"
	desc += fmt.Sprintf("  ├ Name = %s\n", Name())
	desc += fmt.Sprintf("  ├ Version = %s\n", Version())
	desc += fmt.Sprintf("  ├ GitCommit = %s\n", inspectText(GitCommit()))
	desc += fmt.Sprintf("  ├ BuiltBy = %s\n", inspectText(BuiltBy()))
	desc += fmt.Sprintf("  ├ BuiltTime = %s\n", inspectText(BuiltTime()))
	desc += fmt.Sprintf("  ├ GoVersion = %s\n", GoVersion())
	desc += fmt.Sprintf("  ├ GoCompiler = %s\n", GoCompiler())
	desc += fmt.Sprintf("  └ GoPlatform = %s\n", GoPlatform())
	return desc
}

// inspectText reports a build value that the build did not link.
func inspectText(value string) string {
	if value == "" {
		return unavailableText
	}
	return value
}
