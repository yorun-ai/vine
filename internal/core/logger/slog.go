package logger

import (
	"io"
	"log/slog"
)

func newSlogLoggerWithWriter(option WithOption, loggerName string, addSource bool, leveler slog.Leveler, writer io.Writer) *slog.Logger {
	loggerAttr := []slog.Attr{slog.String(loggerKey, loggerName)}
	if option.Format == "" {
		return slog.New(&_GlobalFormatHandler{
			text: newSlogHandler(FormatText, addSource, leveler, writer).WithAttrs(loggerAttr),
			json: newSlogHandler(FormatJSON, addSource, leveler, writer).WithAttrs(loggerAttr),
		})
	}
	return slog.New(newSlogHandler(option.Format, addSource, leveler, writer).WithAttrs(loggerAttr))
}

func newSlogHandler(format Format, addSource bool, leveler slog.Leveler, writer io.Writer) slog.Handler {
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

	switch format {
	case FormatText:
		return slog.NewTextHandler(writer, options)
	case FormatJSON:
		return slog.NewJSONHandler(writer, options)
	default:
		return nil
	}
}
