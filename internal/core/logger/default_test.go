package logger

import (
	"log/slog"
	"testing"
)

func TestDefaultLoggerFunctions(t *testing.T) {
	previousDefault := defaultLogger
	t.Cleanup(func() { SetDefault(previousDefault) })
	Debug("e-ddd")
	Info("e-iii")
	Error("e-eee")

	a := map[string]string{
		"a": "hello",
		"b": "100",
	}
	c := map[string]string{
		"e": "test",
		"f": "100",
	}
	logger := NewLogger(GlobalOption()).With(
		slog.String("a", a["a"]),
		slog.String("b", a["b"]),
		slog.String("e", c["e"]),
		slog.String("f", c["f"]),
	)
	SetDefault(logger)

	Info("test info")
	a["a"] = "world"
	logger.With(
		slog.String("a", a["a"]),
		slog.String("c", "cew"),
		slog.String("d", "999999"),
	).Debug("test debug")
	logger.With(
		slog.String("c", "cew"),
		slog.String("d", "999999"),
	).Error("test error")
}
