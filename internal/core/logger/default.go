package logger

import "go.yorun.ai/vine/util/vpre"

var defaultLogger *Logger

func init() {
	defaultLogger = New()
	setStandardLogger(defaultLogger.config, defaultLogger.leveler)
}

func SetDefault(logger *Logger) {
	vpre.CheckNotNil(logger, "default logger cannot be nil")
	defaultLogger = logger
	setStandardLogger(logger.config, logger.leveler)
}

//go:noinline
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

//go:noinline
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

//go:noinline
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

//go:noinline
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}
