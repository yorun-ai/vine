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
	slog    *slog.Logger
	config  WithOption
	leveler slog.Leveler
}

func New(args ...any) *Logger {
	config := GlobalOption()
	config.Level = LevelAuto
	nameArgs := args
	if len(args) > 0 {
		switch last := args[len(args)-1].(type) {
		case WithOption:
			config = new(last)
			nameArgs = args[:len(args)-1]
		case *WithOption:
			vpre.CheckNotNil(last, "logger WithOption cannot be nil")
			copied := *last
			config = &copied
			nameArgs = args[:len(args)-1]
		}
	}
	vpre.Check(isValidOptionLevel(config.Level), "%+v is not a valid logger option level", config.Level)
	nameSegments := make([]string, 0, len(nameArgs))
	for index, arg := range nameArgs {
		segment, ok := arg.(string)
		vpre.Check(ok, "logger name argument %d must be a string segment; WithOption is allowed only as the last argument", index)
		nameSegments = append(nameSegments, splitNameSegments(segment)...)
	}
	return newLogger(config, newLeveler(config.Level, nameSegments))
}

func splitNameSegments(value string) []string {
	segments := strings.Split(value, ":")
	for _, segment := range segments {
		validWildcard := segment == "*" || segment == "**"
		vpre.Check(segment != "" && (!strings.Contains(segment, "*") || validWildcard),
			"%q is not a valid logger name segment", value)
	}
	return segments
}

func newLogger(config *WithOption, leveler slog.Leveler) *Logger {
	log := new(Logger{
		config:  *config,
		leveler: leveler,
	})
	log.slog = newSlogLogger(config, true, leveler)
	return log
}

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	vpre.Check(len(attrs) > 0, "logger.With requires at least one attr")

	handler := l.slog.Handler().WithAttrs(attrs)
	return new(Logger{
		slog:    slog.New(handler),
		config:  l.config,
		leveler: l.leveler,
	})
}

// Enabled reports whether the logger currently emits records at level.
func (l *Logger) Enabled(level Level) bool {
	return l != nil && l.slog.Handler().Enabled(context.Background(), level.ToSLogLevel())
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
