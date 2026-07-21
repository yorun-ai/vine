package logger

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.yorun.ai/vine/util/vpre"
)

type Logger struct {
	slog   *slog.Logger
	config Option
}

func NewLogger(config *Option) *Logger {
	vpre.CheckNotNil(config, "logger config cannot be nil")
	return &Logger{
		slog:   newSlogLogger(config, true),
		config: *config,
	}
}

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	vpre.Check(len(attrs) > 0, "logger.With requires at least one attr")

	handler := l.slog.Handler().WithAttrs(attrs)
	return &Logger{
		slog:   slog.New(handler),
		config: l.config,
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

// log is the single write path so we can stamp records with the caller PC
// from the external logging call instead of this wrapper layer.
func (l *Logger) log(level Level, msg string, args ...any) {
	slogLevel := level.ToSLogLevel()
	if !l.slog.Handler().Enabled(context.Background(), slogLevel) {
		return
	}

	record := slog.NewRecord(time.Now(), slogLevel, msg, callerPC())
	if len(args) > 0 {
		record.Add(args...)
	}
	_ = l.slog.Handler().Handle(context.Background(), record)
}

// callerPC returns the external logging call frame so source attribution
// points to the caller of this package instead of the wrapper methods here.
func callerPC() uintptr {
	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return 0
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !isLoggerFrame(frame) {
			return frame.PC
		}
		if !more {
			break
		}
	}
	return 0
}

func isLoggerFrame(frame runtime.Frame) bool {
	function := frame.Function
	pkg := trimFunctionPackage(function)
	if pkg == "" {
		return strings.HasPrefix(function, "runtime.") || isLoggerSourceFile(frame.File)
	}
	return pkg == "go.yorun.ai/vine/internal/core/logger" ||
		(pkg == "go.yorun.ai/vine/core/logger" && isFacadeLoggerSourceFile(frame.File)) ||
		pkg == "runtime" ||
		isLoggerSourceFile(frame.File)
}

func isLoggerSourceFile(file string) bool {
	cleanFile := filepath.ToSlash(filepath.Clean(file))
	return strings.Contains(cleanFile, "/internal/core/logger/") && !strings.HasSuffix(cleanFile, "_test.go")
}

func isFacadeLoggerSourceFile(file string) bool {
	cleanFile := filepath.ToSlash(filepath.Clean(file))
	return strings.Contains(cleanFile, "/core/logger/") && !strings.HasSuffix(cleanFile, "_test.go")
}
