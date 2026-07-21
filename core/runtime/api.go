package runtime

import internalruntime "go.yorun.ai/vine/internal/core/runtime"

// App contains the identity of a running Vine application.
type App = internalruntime.App

// Application returns the current process application identity.
func Application() App {
	return internalruntime.Application()
}

// SetName overrides the current application name before it is exposed to the runtime.
func SetName(name string) {
	internalruntime.SetName(name)
}

// SetVersion overrides the current application version.
func SetVersion(version string) {
	internalruntime.SetVersion(version)
}

// SetInstanceId overrides the current application instance identifier.
func SetInstanceId(instanceId string) {
	internalruntime.SetInstanceId(instanceId)
}

// Inspect returns a human-readable summary of application and build metadata.
func Inspect() string {
	return internalruntime.Inspect()
}

// GitCommit returns the commit recorded at build time, or an empty string when unavailable.
func GitCommit() string {
	return internalruntime.GitCommit()
}

// BuiltBy returns the builder recorded at build time, or an empty string when unavailable.
func BuiltBy() string {
	return internalruntime.BuiltBy()
}

// BuiltTime returns the build timestamp, or an empty string when unavailable.
func BuiltTime() string {
	return internalruntime.BuiltTime()
}

// GolangVersion returns the Go version used to build the executable.
func GolangVersion() string {
	return internalruntime.GoVersion()
}

// GolangCompiler returns the compiler used to build the executable.
func GolangCompiler() string {
	return internalruntime.GoCompiler()
}

// GolangPlatform returns the target operating system and architecture of the executable.
func GolangPlatform() string {
	return internalruntime.GoPlatform()
}
