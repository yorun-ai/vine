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

// GlobalOption returns a copy of the process-wide logging configuration.
func GlobalOption() *Option {
	return internallogger.GlobalOption()
}

// SetGlobalMode changes the process-wide output mode.
func SetGlobalMode(mode Mode) {
	internallogger.SetGlobalMode(mode)
}

// SetGlobalLevel changes the process-wide minimum level.
func SetGlobalLevel(level Level) {
	internallogger.SetGlobalLevel(level)
}

// NewLogger creates a logger from config.
func NewLogger(config *Option) *Logger {
	return internallogger.NewLogger(config)
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
