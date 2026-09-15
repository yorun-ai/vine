package logger

import (
	"context"
	"encoding/json/v2"
	stdLog "log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	wait.Go(func() {
		for index := range 100 {
			if index%2 == 0 {
				SetDefault(first)
			} else {
				SetDefault(second)
			}
		}
	})
	wait.Go(func() {
		for index := range 100 {
			if index%2 == 0 {
				SetDefault(second)
			} else {
				SetDefault(first)
			}
		}
	})
	wait.Go(func() {
		for range 100 {
			Debug("concurrent-default-log")
			stdLog.Print("concurrent-standard-log")
		}
	})
	wait.Wait()
}

func TestDefaultLoggerFunctions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default-functions.jsonl")
	previousDefault := defaultLoggers.Load().logger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetDefault(New("vine:test", WithOption{
		Format:     FormatJSON,
		Level:      LevelDebug,
		OutputPath: path,
	}))

	Debug("test debug")
	Info("test info")
	Error("test error")

	want := []struct {
		level   string
		message string
	}{
		{level: "DEBUG", message: "test debug"},
		{level: "INFO", message: "test info"},
		{level: "ERROR", message: "test error"},
	}
	records := readDefaultLoggerRecords(t, path)
	if len(records) != len(want) {
		t.Fatalf("default logger records = %#v, want %d records", records, len(want))
	}
	for index, wantRecord := range want {
		got := records[index]
		if got.Level != wantRecord.level || got.Logger != "vine:test" || got.Message != wantRecord.message {
			t.Fatalf("record %d = %#v, want level %s logger vine:test message %q", index, got, wantRecord.level, wantRecord.message)
		}
	}
}

type defaultLoggerRecord struct {
	Level   string `json:"level"`
	Logger  string `json:"logger"`
	Message string `json:"msg"`
}

func readDefaultLoggerRecords(t *testing.T, path string) []defaultLoggerRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read default logger output: %v", err)
	}
	records := make([]defaultLoggerRecord, 0, 4)
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var record defaultLoggerRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode default logger record: %v", err)
		}
		records = append(records, record)
	}
	return records
}
