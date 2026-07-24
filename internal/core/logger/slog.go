package logger

import (
	"io"
	stdLog "log"
	"log/slog"
	"os"

	"go.yorun.ai/vine/util/vpre"
)

// newSlogLogger builds the underlying slog logger using the configured output mode,
// level, source handling, and optional file mirroring.
func newSlogLogger(config *Option, addSource bool, leveler slog.Leveler) *slog.Logger {
	options := &slog.HandlerOptions{
		Level:     leveler,
		AddSource: addSource,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.SourceKey {
				return attr
			}

			source, ok := attr.Value.Any().(*slog.Source)
			if !ok || source == nil {
				return attr
			}
			copied := *source
			copied.File = trimSourceFile(&copied)
			return slog.Any(attr.Key, &copied)
		},
	}

	writer := newLogWriter(config.OutputPath)
	vpre.Check(IsValidMode(config.Mode), "%+v is not a valid LogMode", config.Mode)
	switch config.Mode {
	case ModeText:
		return slog.New(slog.NewTextHandler(writer, options))
	case ModeJSON:
		return slog.New(slog.NewJSONHandler(writer, options))
	default:
		return nil
	}
}

// newLogWriter returns a process-lifetime writer for the logger output.
// When OutputPath is set, the opened file is intentionally kept for the
// lifetime of the process and is not closed by the logger package.
func newLogWriter(outputPath string) io.Writer {
	writer := io.Writer(os.Stderr)
	if outputPath == "" {
		return writer
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		stdLog.Fatal(err)
	}
	return io.MultiWriter(os.Stderr, file)
}
