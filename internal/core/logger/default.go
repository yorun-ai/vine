package logger

import (
	"sync/atomic"

	"go.yorun.ai/vine/util/vpre"
)

var defaultLogger atomic.Pointer[Logger]

func init() {
	logger := New("vine:default")
	defaultLogger.Store(logger)
	setStandardLogger(logger)
}

func SetDefault(logger *Logger) {
	vpre.CheckNotNil(logger, "default logger cannot be nil")
	defaultLogger.Store(logger)
	setStandardLogger(logger)
}

//go:noinline
func Debug(msg string, args ...any) {
	defaultLogger.Load().Debug(msg, args...)
}

//go:noinline
func Info(msg string, args ...any) {
	defaultLogger.Load().Info(msg, args...)
}

//go:noinline
func Warn(msg string, args ...any) {
	defaultLogger.Load().Warn(msg, args...)
}

//go:noinline
func Error(msg string, args ...any) {
	defaultLogger.Load().Error(msg, args...)
}
