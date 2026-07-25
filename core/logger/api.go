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

// Level is either a concrete logging threshold or the LevelAuto policy used
// by WithOption.
type Level = internallogger.Level

const (
	// LevelAuto dynamically resolves the logging threshold from matching named
	// rules and falls back to the process-wide global level.
	LevelAuto = internallogger.LevelAuto
	// LevelDebug enables debug and higher-severity records.
	LevelDebug = internallogger.LevelDebug
	// LevelInfo enables informational and higher-severity records.
	LevelInfo = internallogger.LevelInfo
	// LevelWarn enables warning and error records.
	LevelWarn = internallogger.LevelWarn
	// LevelError enables only error records.
	LevelError = internallogger.LevelError
)

// WithOption configures the logger created by New. When supplied, its value or
// pointer must be the final New argument.
type WithOption = internallogger.WithOption

// Logger writes structured log records.
type Logger = internallogger.Logger

// IsValidLevel reports whether level is a supported logging threshold.
func IsValidLevel(level Level) bool {
	return internallogger.IsValidLevel(level)
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
func GlobalOption() *WithOption {
	return internallogger.GlobalOption()
}

// SetGlobalMode changes the output mode used by loggers created afterward.
func SetGlobalMode(mode Mode) {
	internallogger.SetGlobalMode(mode)
}

// SetGlobalLevel changes the fallback threshold for auto-level loggers that
// match no named level rule.
func SetGlobalLevel(level Level) {
	internallogger.SetGlobalLevel(level)
}

// New creates a logger whose required name and additional name arguments are
// joined with ":".
// Each name argument may itself contain colon-separated segments.
// A WithOption value or pointer may be supplied only as the final argument.
// Without WithOption, the logger uses the current global mode and dynamically
// follows the current pattern-to-level configuration.
func New(name string, args ...any) *Logger {
	return internallogger.New(name, args...)
}

// SetLevel sets a process-local logging threshold for pattern. Names and
// patterns are colon-separated. A whole-segment "*" matches exactly one
// segment, while "**" matches zero or more consecutive segments. A matching
// pattern also applies to descendant names. When several rules match, literal
// segments outrank "*", "*" outranks "**", and comparison proceeds from left
// to right before longer patterns outrank their prefixes. Pure wildcard
// patterns "*" and "**" are reserved.
func SetLevel(pattern string, level Level) {
	internallogger.SetLevel(pattern, level)
}

// ClearLevel removes the logging threshold configured for pattern.
func ClearLevel(pattern string) {
	internallogger.ClearLevel(pattern)
}

// Levels returns a copy of the current pattern-to-level configuration.
func Levels() map[string]Level {
	return internallogger.Levels()
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
