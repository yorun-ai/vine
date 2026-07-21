package buildinfo

// This file exposes ldflags-injected runtime identity for applications that
// depend on the Vine framework. The internal core runtime uses these values as
// application metadata for registration, RPC/event/task headers, and other
// framework runtime behavior.

var (
	// WILL BE MODIFIED BY LDFLAGS when building
	ldName      = defaultName
	ldVersion   = defaultVersion // must be semver
	ldGitCommit = defaultBuildText
	ldBuiltBy   = defaultBuildText
	ldBuiltTime = defaultBuildText
)

const (
	defaultName      = "vined"
	defaultVersion   = "0.0.0"
	defaultBuildText = "NotAvailable"
)

// Name returns the linker-injected executable name and whether it differs from the default.
func Name() (string, bool) {
	return ldName, ldName != defaultName
}

// Version returns the linker-injected version and whether it differs from the default.
func Version() (string, bool) {
	return ldVersion, ldVersion != defaultVersion
}

// GitCommit returns the linker-injected commit and whether it is available.
func GitCommit() (string, bool) {
	return ldGitCommit, ldGitCommit != defaultBuildText
}

// BuiltBy returns the linker-injected builder and whether it is available.
func BuiltBy() (string, bool) {
	return ldBuiltBy, ldBuiltBy != defaultBuildText
}

// BuiltTime returns the linker-injected build time and whether it is available.
func BuiltTime() (string, bool) {
	return ldBuiltTime, ldBuiltTime != defaultBuildText
}
