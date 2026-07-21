package logger_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	logger "go.yorun.ai/vine/internal/core/logger"
)

type loggedSource struct {
	File string `json:"file"`
}

type loggedRecord struct {
	Level   string       `json:"level"`
	Message string       `json:"msg"`
	Source  loggedSource `json:"source"`
}

func TestLoggerInfoUsesExternalCallerSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logger.jsonl")
	log := logger.NewLogger(&logger.Option{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	})

	logFromHelper(log)

	record := readLastRecord(t, path)
	if record.Message != "helper-log" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if !strings.HasSuffix(record.Source.File, "source_test.go") {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
	if strings.Contains(record.Source.File, "asm_") {
		t.Fatalf("unexpected runtime source file: %q", record.Source.File)
	}
}

func TestDefaultLoggerInfoUsesExternalCallerSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default-logger.jsonl")
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	logFromDefaultHelper()

	record := readLastRecord(t, path)
	if record.Message != "default-helper-log" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if !strings.HasSuffix(record.Source.File, "source_test.go") {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
	if strings.Contains(record.Source.File, "default.go") {
		t.Fatalf("unexpected logger wrapper source file: %q", record.Source.File)
	}
}

func TestDefaultAndCustomLoggerUseSameCallerSource(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom.jsonl")
	defaultPath := filepath.Join(t.TempDir(), "default.jsonl")

	customLogger := logger.NewLogger(&logger.Option{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: customPath,
	})
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: defaultPath,
	}))
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	logBothFromHelper(customLogger)

	customRecord := readLastRecord(t, customPath)
	defaultRecord := readLastRecord(t, defaultPath)
	if customRecord.Source.File != defaultRecord.Source.File {
		t.Fatalf("unexpected source file mismatch: custom=%q default=%q", customRecord.Source.File, defaultRecord.Source.File)
	}
}

func logFromHelper(log *logger.Logger) {
	log.Info("helper-log")
}

func logFromDefaultHelper() {
	logger.Info("default-helper-log")
}

func logBothFromHelper(log *logger.Logger) {
	log.Info("helper-log")
	logger.Info("default-helper-log")
}

func readLastRecord(t *testing.T, path string) loggedRecord {
	t.Helper()

	records := readAllRecords(t, path)
	return records[len(records)-1]
}

func readAllRecords(t *testing.T, path string) []loggedRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected at least one log line")
	}

	records := make([]loggedRecord, 0, len(lines))
	for _, line := range lines {
		var record loggedRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("unmarshal log record: %v", err)
		}
		records = append(records, record)
	}
	return records
}
