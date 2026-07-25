package logger

import (
	"context"
	stdLog "log"
	"log/slog"
	"sync"
	"testing"
)

func TestDefaultLoggerName(t *testing.T) {
	if got := defaultLoggers.Load().logger.Name(); got != "vine:default" {
		t.Fatalf("default logger name = %q, want %q", got, "vine:default")
	}
}

func TestStandardLoggerUsesIndependentName(t *testing.T) {
	resetRulesForTest(t)
	previousDefault := defaultLoggers.Load().logger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetGlobalLevel(LevelError)
	SetDefault(New("vine:default", WithOption{Format: FormatText, Level: LevelAuto}))
	SetLevel("vine:stdlog", LevelDebug)

	if defaultLoggers.Load().logger.Enabled(LevelDebug) {
		t.Fatal("default logger should continue using the global fallback")
	}
	if !defaultLoggers.Load().standard.Handler().Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("standard logger should resolve levels using vine:stdlog")
	}
}

func TestDefaultLoggerKeepsInjectedLoggerLevelSemantics(t *testing.T) {
	resetGlobalOptionForTest(t)
	previousDefault := defaultLoggers.Load().logger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetGlobalLevel(LevelInfo)

	SetDefault(New("vine:test", WithOption{Format: FormatText, Level: LevelAuto}))
	SetGlobalLevel(LevelDebug)
	if !defaultLoggers.Load().logger.Enabled(LevelDebug) {
		t.Fatal("default auto logger should follow the default level")
	}

	fixed := New("vine:test", WithOption{Format: FormatText, Level: LevelInfo})
	SetDefault(fixed)
	if defaultLoggers.Load().logger.Enabled(LevelDebug) {
		t.Fatal("explicit fixed default logger should keep its own threshold")
	}
}

func TestSetDefaultConcurrentLogging(t *testing.T) {
	previousDefault := defaultLoggers.Load().logger
	t.Cleanup(func() { SetDefault(previousDefault) })

	first := New("vine:test:first", WithOption{Format: FormatText, Level: LevelError})
	second := New("vine:test:second", WithOption{Format: FormatText, Level: LevelError})
	var wait sync.WaitGroup
	wait.Add(3)
	go func() {
		defer wait.Done()
		for index := range 100 {
			if index%2 == 0 {
				SetDefault(first)
			} else {
				SetDefault(second)
			}
		}
	}()
	go func() {
		defer wait.Done()
		for index := range 100 {
			if index%2 == 0 {
				SetDefault(second)
			} else {
				SetDefault(first)
			}
		}
	}()
	go func() {
		defer wait.Done()
		for range 100 {
			Debug("concurrent-default-log")
			stdLog.Print("concurrent-standard-log")
		}
	}()
	wait.Wait()
}

func TestDefaultLoggerFunctions(t *testing.T) {
	previousDefault := defaultLoggers.Load().logger
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
	logger := New("vine:test").With(
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
