package logger

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestDefaultLoggerName(t *testing.T) {
	if got := strings.Join(defaultLogger.nameSegments, ":"); got != "vine:default" {
		t.Fatalf("default logger name = %q, want %q", got, "vine:default")
	}
}

func TestStandardLoggerUsesIndependentName(t *testing.T) {
	resetRulesForTest(t)
	previousDefault := defaultLogger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetGlobalLevel(LevelError)
	SetDefault(New("vine:default", WithOption{Mode: ModeText, Level: LevelAuto}))
	SetLevel("vine:stdlog", LevelDebug)

	if defaultLogger.Enabled(LevelDebug) {
		t.Fatal("default logger should continue using the global fallback")
	}
	if !standardLogger.Handler().Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("standard logger should resolve levels using vine:stdlog")
	}
}

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
	logger := New("vine:test", GlobalOption()).With(
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
