package logger

import internallogger "go.yorun.ai/vine/internal/core/logger"

// Format controls the output encoding of log records.
type Format = internallogger.Format

const (
	// FormatJSON emits one JSON object per line.
	FormatJSON = internallogger.FormatJSON
	// FormatText emits one line of human-readable key=value pairs per record.
	FormatText = internallogger.FormatText
)

// IsValidFormat reports whether format is a supported output format.
func IsValidFormat(format Format) bool {
	return internallogger.IsValidFormat(format)
}

// Level is either a concrete logging threshold or the LevelAuto policy used
// by WithOption.
type Level = internallogger.Level

const (
	// LevelAuto is the zero value. It dynamically resolves the logging threshold
	// from matching named rules and falls back to the process-wide global level.
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

// WithOption configures the logger created by New. An empty Format or
// OutputPath dynamically follows its process-wide global setting, and an empty
// Level is LevelAuto. When supplied, the option must be the final New argument.
type WithOption = internallogger.WithOption

// Logger writes structured log records. Every record includes the reserved
// "logger" field containing its complete colon-separated name.
type Logger = internallogger.Logger

// IsValidLevel reports whether level is a supported logging threshold.
func IsValidLevel(level Level) bool {
	return internallogger.IsValidLevel(level)
}

// SetGlobalFormat changes the format used by loggers that inherit the global
// output format, including existing loggers.
func SetGlobalFormat(format Format) {
	internallogger.SetGlobalFormat(format)
}

// SetGlobalOutputPath changes the mirrored file used by loggers that inherit
// the global output. An empty path writes only to stderr.
func SetGlobalOutputPath(outputPath string) {
	internallogger.SetGlobalOutputPath(outputPath)
}

// SetGlobalLevel changes the fallback threshold for auto-level loggers that
// match no named level rule.
func SetGlobalLevel(level Level) {
	internallogger.SetGlobalLevel(level)
}

// New creates a logger whose required name and additional name arguments are
// joined with ":".
// Each name argument may itself contain colon-separated segments.
// A WithOption value may be supplied only as the final argument.
// Without WithOption, the logger dynamically follows the global format,
// global output path, and current pattern-to-level configuration.
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
