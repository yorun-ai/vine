package logger

import internallogger "go.yorun.ai/vine/internal/core/logger"

// Mode controls the output encoding of log records.
type Mode = internallogger.Mode

const (
	// ModeJSON emits one JSON object per log record.
	ModeJSON = internallogger.ModeJSON
	// ModeText emits human-readable text records.
	ModeText = internallogger.ModeText
)

// Level is the minimum severity emitted by a logger.
type Level = internallogger.Level

const (
	// LevelDebug enables debug and higher-severity records.
	LevelDebug = internallogger.LevelDebug
	// LevelInfo enables informational and higher-severity records.
	LevelInfo = internallogger.LevelInfo
	// LevelWarn enables warning and error records.
	LevelWarn = internallogger.LevelWarn
	// LevelError enables only error records.
	LevelError = internallogger.LevelError
)

// Option configures a Logger.
type Option = internallogger.Option

// Logger writes structured log records.
type Logger = internallogger.Logger

// Subsystem identifies a framework logging scope.
type Subsystem = internallogger.Subsystem

const (
	// SubsystemRpcServer identifies Rpc server lifecycle logs.
	SubsystemRpcServer = internallogger.SubsystemRpcServer
	// SubsystemTask identifies task launcher and runner logs.
	SubsystemTask = internallogger.SubsystemTask
	// SubsystemEvent identifies event emitter and listener logs.
	SubsystemEvent = internallogger.SubsystemEvent
)

// Scope identifies the local application and framework subsystem that own a logger.
type Scope = internallogger.Scope

// AppSubsystemLevel defines one App plus subsystem threshold override.
type AppSubsystemLevel = internallogger.AppSubsystemLevel

// LevelOverrides is a complete replacement input for scoped threshold overrides.
type LevelOverrides = internallogger.LevelOverrides

// IsValidLevel reports whether level is a supported logging threshold.
func IsValidLevel(level Level) bool {
	return internallogger.IsValidLevel(level)
}

// IsValidSubsystem reports whether subsystem is a supported framework scope.
func IsValidSubsystem(subsystem Subsystem) bool {
	return internallogger.IsValidSubsystem(subsystem)
}

// PayloadSurface identifies a lifecycle payload field.
type PayloadSurface = internallogger.PayloadSurface

const (
	// PayloadSurfaceRpcArguments identifies Rpc argument payloads.
	PayloadSurfaceRpcArguments = internallogger.PayloadSurfaceRpcArguments
	// PayloadSurfaceRpcResult identifies Rpc result payloads.
	PayloadSurfaceRpcResult = internallogger.PayloadSurfaceRpcResult
	// PayloadSurfaceEvent identifies Event listener payloads.
	PayloadSurfaceEvent = internallogger.PayloadSurfaceEvent
)

// PayloadMode controls whether and how a lifecycle payload is redacted.
type PayloadMode = internallogger.PayloadMode

const (
	// PayloadModeSafe applies built-in sensitive-key redaction.
	PayloadModeSafe = internallogger.PayloadModeSafe
	// PayloadModeOff disables payload logging for a selector.
	PayloadModeOff = internallogger.PayloadModeOff
	// PayloadModeUnsafeFull skips sensitive-key redaction for an exact selector.
	PayloadModeUnsafeFull = internallogger.PayloadModeUnsafeFull
)

// PayloadDescriptor describes the contract location of one payload.
type PayloadDescriptor = internallogger.PayloadDescriptor

// PayloadSanitizer converts a contract payload into a safe projection before logging.
// A sanitizer must be side-effect free and must not mutate the supplied payload.
type PayloadSanitizer = internallogger.PayloadSanitizer

// PayloadPolicy configures redaction and optional sanitization for a payload selector.
// Payload size and traversal budgets are not part of the current API.
type PayloadPolicy = internallogger.PayloadPolicy

// GlobalOption returns a copy of the process-wide logging configuration.
func GlobalOption() *Option {
	return internallogger.GlobalOption()
}

// SetGlobalMode changes the output mode used by loggers created afterward.
func SetGlobalMode(mode Mode) {
	internallogger.SetGlobalMode(mode)
}

// SetGlobalLevel changes the threshold followed by existing and future global and scoped loggers.
func SetGlobalLevel(level Level) {
	internallogger.SetGlobalLevel(level)
}

// NewLogger creates a fixed-level logger from config.
// It does not follow later global or scoped threshold changes.
func NewLogger(config *Option) *Logger {
	return internallogger.NewLogger(config)
}

// NewGlobalLogger creates a logger that follows later SetGlobalLevel calls.
func NewGlobalLogger() *Logger {
	return internallogger.NewGlobalLogger()
}

// NewScopedLogger creates a logger that dynamically resolves App plus subsystem, App,
// subsystem, and global thresholds in that order.
func NewScopedLogger(scope Scope) *Logger {
	return internallogger.NewScopedLogger(scope)
}

// SetSubsystemLevel sets a process-local subsystem threshold override.
func SetSubsystemLevel(subsystem Subsystem, level Level) {
	internallogger.SetSubsystemLevel(subsystem, level)
}

// ClearSubsystemLevel removes a process-local subsystem threshold override.
func ClearSubsystemLevel(subsystem Subsystem) {
	internallogger.ClearSubsystemLevel(subsystem)
}

// SetAppLevel sets a process-local application threshold override.
func SetAppLevel(appName string, level Level) {
	internallogger.SetAppLevel(appName, level)
}

// ClearAppLevel removes a process-local application threshold override.
func ClearAppLevel(appName string) {
	internallogger.ClearAppLevel(appName)
}

// SetAppSubsystemLevel sets a process-local App plus subsystem threshold override.
func SetAppSubsystemLevel(appName string, subsystem Subsystem, level Level) {
	internallogger.SetAppSubsystemLevel(appName, subsystem, level)
}

// ClearAppSubsystemLevel removes a process-local App plus subsystem threshold override.
func ClearAppSubsystemLevel(appName string, subsystem Subsystem) {
	internallogger.ClearAppSubsystemLevel(appName, subsystem)
}

// ReplaceLevelOverrides validates and atomically replaces all scoped overrides.
func ReplaceLevelOverrides(overrides LevelOverrides) {
	internallogger.ReplaceLevelOverrides(overrides)
}

// RegisterRpcPayloadPolicy registers a policy for one Rpc payload surface before an App or server starts.
func RegisterRpcPayloadPolicy(serviceSkelName string, methodSkelName string, surface PayloadSurface, policy PayloadPolicy) {
	internallogger.RegisterRpcPayloadPolicy(serviceSkelName, methodSkelName, surface, policy)
}

// RegisterEventPayloadPolicy registers a policy for one Event payload before an App or server starts.
func RegisterEventPayloadPolicy(eventSkelName string, policy PayloadPolicy) {
	internallogger.RegisterEventPayloadPolicy(eventSkelName, policy)
}

// RegisterPayloadSurfacePolicy registers a default for one payload surface before an App or server starts.
// Unsafe-full mode requires an exact Rpc method or Event selector and is rejected here.
func RegisterPayloadSurfacePolicy(surface PayloadSurface, policy PayloadPolicy) {
	internallogger.RegisterPayloadSurfacePolicy(surface, policy)
}

// SetDefault replaces the package-level logger used by Debug, Info, Warn, and Error.
func SetDefault(logger *Logger) {
	internallogger.SetDefault(logger)
}

// Debug writes a structured debug record.
func Debug(msg string, args ...any) {
	internallogger.Debug(msg, args...)
}

// Info writes a structured informational record.
func Info(msg string, args ...any) {
	internallogger.Info(msg, args...)
}

// Warn writes a structured warning record.
func Warn(msg string, args ...any) {
	internallogger.Warn(msg, args...)
}

// Error writes a structured error record.
func Error(msg string, args ...any) {
	internallogger.Error(msg, args...)
}
