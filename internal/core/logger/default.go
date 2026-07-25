package logger

import (
	"log/slog"
	"sync/atomic"

	"go.yorun.ai/vine/util/vpre"
)

type _DefaultLoggers struct {
	logger   *Logger
	standard *slog.Logger
}

var defaultLoggers atomic.Pointer[_DefaultLoggers]

func init() { SetDefault(New("vine:default")) }

func SetDefault(logger *Logger) {
	vpre.CheckNotNil(logger, "default logger cannot be nil")
	defaultLoggers.Store(&_DefaultLoggers{
		logger:   logger,
		standard: newStandardLogger(logger),
	})
}

//go:noinline
func Debug(msg string, args ...any) {
	defaultLoggers.Load().logger.Debug(msg, args...)
}

//go:noinline
func Info(msg string, args ...any) {
	defaultLoggers.Load().logger.Info(msg, args...)
}

//go:noinline
func Warn(msg string, args ...any) {
	defaultLoggers.Load().logger.Warn(msg, args...)
}

//go:noinline
func Error(msg string, args ...any) {
	defaultLoggers.Load().logger.Error(msg, args...)
}
