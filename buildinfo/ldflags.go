package buildinfo

// This file exposes the values a build links into a binary that depends on the
// Vine framework: the executable name, the version applications report, and the
// commit, builder, and build time of the build.
//
// Every variable below is replaced with
// -X go.yorun.ai/vine/buildinfo.<name>=<value> when the binary is built. A
// build that links no value keeps the corresponding default; the getters in
// info.go report a default as unavailable.

// Defaults for the values a build does not link. The remaining variables default
// to the empty string.
const (
	defaultName    = "vined"
	defaultVersion = "0.0.0"
)

var (
	// ldName is the executable name, injected as ldName. Dot-separated segments
	// carry lowercase letters and digits with dashes between them, for example
	// user.service or demo.worker-2; an unusable name panics when the process
	// starts. Default: "vined".
	ldName = defaultName

	// ldVersion is the application version, injected as ldVersion. It must be a
	// full semantic version and may carry the Go module "v" prefix, for example
	// v1.2.3; an unusable version panics when Version reads it.
	// Default: "0.0.0".
	ldVersion = defaultVersion

	// ldGitCommit identifies the commit the binary was built from, injected as
	// ldGitCommit. It carries no format requirement; Plot links the short SHA
	// and marks a dirty workspace with a "-dirty" suffix. Empty when the build
	// links no commit.
	ldGitCommit string

	// ldBuiltBy names the tool that built the binary, injected as ldBuiltBy. It
	// carries no format requirement; Plot links plot/<version>. Empty when the
	// build links no builder.
	ldBuiltBy string

	// ldBuiltTime records when the binary was built, injected as ldBuiltTime.
	// It carries no format requirement. Empty when the build links no time.
	ldBuiltTime string
)
