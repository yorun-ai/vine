package logger

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.yorun.ai/vine/util/vpre"
)

// Option

// WithOption configures the logger created by New.
type WithOption struct {
	// An empty Format follows the process-wide global format.
	Format Format
	// An empty Level is LevelAuto.
	Level Level
	// An empty OutputPath follows the process-wide global output.
	OutputPath string
}

// Logger

// Logger writes structured records containing the reserved "logger" field
// with its complete colon-separated name.
type Logger struct {
	slog         *slog.Logger
	option       WithOption
	nameSegments []string
	attrs        []slog.Attr
	writer       io.Writer
}

func New(name string, args ...any) *Logger {
	var option WithOption
	nameArgs := make([]any, 0, len(args)+1)
	nameArgs = append(nameArgs, name)
	nameArgs = append(nameArgs, args...)
	if len(args) > 0 {
		switch last := args[len(args)-1].(type) {
		case WithOption:
			option = last
			nameArgs = nameArgs[:len(nameArgs)-1]
		}
	}
	vpre.Check(isValidOptionLevel(option.Level), "%+v is not a valid logger option level", option.Level)
	vpre.Check(isValidOptionFormat(option.Format), "%+v is not a valid logger option format", option.Format)
	nameSegments := make([]string, 0, len(nameArgs))
	for index, arg := range nameArgs {
		segment, ok := arg.(string)
		vpre.Check(ok, "logger name argument %d must be a string segment; WithOption is allowed only as the last argument", index)
		nameSegments = append(nameSegments, splitNameSegments(segment)...)
	}
	writer := io.Writer(globalWriter)
	if option.OutputPath != "" {
		writer = sharedLogWriter(option.OutputPath)
	}
	return newLogger(option, nameSegments, nil, writer)
}

// Name

func splitNameSegments(value string) []string {
	segments := strings.Split(value, ":")
	for _, segment := range segments {
		vpre.Check(segment != "" && !strings.Contains(segment, "*"),
			"%q is not a valid logger name", value)
	}
	return segments
}

func newLogger(option WithOption, nameSegments []string, attrs []slog.Attr, writer io.Writer) *Logger {
	leveler := newLeveler(option.Level, nameSegments)
	slogLogger := newSlogLoggerWithWriter(option, strings.Join(nameSegments, ":"), true, leveler, writer)
	if len(attrs) > 0 {
		slogLogger = slog.New(slogLogger.Handler().WithAttrs(attrs))
	}
	return &Logger{
		slog:         slogLogger,
		option:       option,
		nameSegments: append([]string(nil), nameSegments...),
		attrs:        append([]slog.Attr(nil), attrs...),
		writer:       writer,
	}
}

// Derived loggers

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	vpre.Check(len(attrs) > 0, "logger.With requires at least one attr")

	handler := l.slog.Handler().WithAttrs(attrs)
	return &Logger{
		slog:         slog.New(handler),
		option:       l.option,
		nameSegments: append([]string(nil), l.nameSegments...),
		attrs:        append(append([]slog.Attr(nil), l.attrs...), attrs...),
		writer:       l.writer,
	}
}

// Child returns a logger whose name extends this logger's name.
// Options and attributes are inherited from the parent.
func (l *Logger) Child(name string, names ...string) *Logger {
	nameSegments := append([]string(nil), l.nameSegments...)
	nameSegments = append(nameSegments, splitNameSegments(name)...)
	for _, next := range names {
		nameSegments = append(nameSegments, splitNameSegments(next)...)
	}
	return newLogger(l.option, nameSegments, l.attrs, l.writer)
}

// Logging

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

// Source

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
