package conf

import (
	internalconf "go.yorun.ai/vine/internal/core/conf"
)

// Lifecycle controls whether a configuration is retained or consumed after it is read.
type Lifecycle = internalconf.Lifecycle

// Config is implemented by generated and application-defined configuration models.
type Config = internalconf.Config

// Reader resolves registered configurations by name or Go type.
type Reader = internalconf.Reader

// ConfigModel can be embedded to implement Config.
type ConfigModel = internalconf.ConfigModel

// ConfigSpec describes a configuration's registered name, type, and lifecycle.
type ConfigSpec = internalconf.ConfigSpec

const (
	// LifecycleEternal retains configuration for repeated reads.
	LifecycleEternal Lifecycle = internalconf.LifecycleEternal
	// LifecycleInstant consumes configuration according to the runtime's instant lifecycle rules.
	LifecycleInstant Lifecycle = internalconf.LifecycleInstant
)

// ParseLifecycle parses a lifecycle name and reports whether it is valid.
func ParseLifecycle(s string) (Lifecycle, bool) {
	return internalconf.ParseLifecycle(s)
}

// Register adds spec to the process-wide configuration registry.
func Register(spec ConfigSpec) {
	internalconf.Register(spec)
}
