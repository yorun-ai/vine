package logger

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"

	"go.yorun.ai/vine/util/vpre"
)

// Format

type Format string

const (
	// FormatJSON emits one JSON object per line.
	FormatJSON Format = "JSON"
	// FormatText emits one line of human-readable key=value pairs per record.
	FormatText Format = "TEXT"
)

func IsValidFormat(format Format) bool {
	return format == FormatJSON || format == FormatText
}

func isValidOptionFormat(format Format) bool {
	return format == "" || IsValidFormat(format)
}

// Global format

var globalFormat = newGlobalFormat()

func newGlobalFormat() *atomic.Pointer[Format] {
	format := FormatText
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		format = FormatJSON
	}

	value := new(atomic.Pointer[Format])
	value.Store(&format)
	return value
}

func SetGlobalFormat(format Format) {
	vpre.Check(IsValidFormat(format), "%+v is not a valid log format", format)
	globalFormat.Store(&format)
}

func currentGlobalFormat() Format {
	return *globalFormat.Load()
}

// Global format handler

type _GlobalFormatHandler struct {
	text slog.Handler
	json slog.Handler
}

func (h *_GlobalFormatHandler) current() slog.Handler {
	if currentGlobalFormat() == FormatJSON {
		return h.json
	}
	return h.text
}

func (h *_GlobalFormatHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.current().Enabled(ctx, level)
}

func (h *_GlobalFormatHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.current().Handle(ctx, record)
}

func (h *_GlobalFormatHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &_GlobalFormatHandler{
		text: h.text.WithAttrs(attrs),
		json: h.json.WithAttrs(attrs),
	}
}

func (h *_GlobalFormatHandler) WithGroup(name string) slog.Handler {
	return &_GlobalFormatHandler{
		text: h.text.WithGroup(name),
		json: h.json.WithGroup(name),
	}
}
