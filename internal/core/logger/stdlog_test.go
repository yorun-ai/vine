package logger_test

import (
	"encoding/json"
	"log"
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
	log := logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	})

	logFromHelper(log)

	record := readLastRecord(t, path)
	if record.Message != "helper-log" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if !strings.HasSuffix(record.Source.File, "stdlog_test.go") {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
	if strings.Contains(record.Source.File, "asm_") {
		t.Fatalf("unexpected runtime source file: %q", record.Source.File)
	}
}

func TestDefaultLoggerInfoUsesExternalCallerSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default-logger.jsonl")
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
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
	if !strings.HasSuffix(record.Source.File, "stdlog_test.go") {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
	if strings.Contains(record.Source.File, "default.go") {
		t.Fatalf("unexpected logger wrapper source file: %q", record.Source.File)
	}
}

func TestDefaultAndCustomLoggerUseSameCallerSource(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom.jsonl")
	defaultPath := filepath.Join(t.TempDir(), "default.jsonl")

	customLogger := logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: customPath,
	})
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
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

func TestStandardLogWritesThroughDefaultLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog.jsonl")
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	log.Println("stdlog-bridge")

	records := readAllRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("unexpected record count: %d", len(records))
	}
	if records[0].Level != "DEBUG" || records[0].Message != "stdlog-bridge" || records[0].Source.File != "STDLOG" {
		t.Fatalf("unexpected debug record: %+v", records[0])
	}
	record := records[1]
	if record.Message != "stdlog-bridge" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if record.Level != "INFO" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Source.File != "STDLOG" {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
}

func TestStandardLogProcessorDropsMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog-drop.jsonl")
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	logger.ConfigureStdLogProcessors("drop-test", func(msg string) (string, bool, bool) {
		if msg == "drop-me-once" {
			return "", false, true
		}
		return "", false, false
	})
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	log.Println("drop-me-once")

	records := readAllRecords(t, path)
	if len(records) != 1 {
		t.Fatalf("unexpected record count: %d", len(records))
	}
	record := records[0]
	if record.Level != "DEBUG" || record.Message != "drop-me-once" || record.Source.File != "STDLOG" {
		t.Fatalf("unexpected raw debug record: %+v", record)
	}
}

func TestStandardLogProcessorRewritesMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog-rewrite.jsonl")
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	logger.ConfigureStdLogProcessors("rewrite-test", func(msg string) (string, bool, bool) {
		if msg == "rewrite-me-once" {
			return "[stdlog] " + msg, true, true
		}
		return "", false, false
	})
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	log.Println("rewrite-me-once")

	records := readAllRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("unexpected record count: %d", len(records))
	}
	if records[0].Level != "DEBUG" || records[0].Message != "rewrite-me-once" || records[0].Source.File != "STDLOG" {
		t.Fatalf("unexpected debug record: %+v", records[0])
	}
	record := records[1]
	if record.Message != "[stdlog] rewrite-me-once" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if record.Level != "INFO" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Source.File != "STDLOG/rewrite-test" {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
}

func TestStandardLogProcessorStopsAtFirstMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog-first-match.jsonl")
	original := logger.New(logger.GlobalOption())
	logger.SetDefault(logger.New(logger.WithOption{
		Mode:       logger.ModeJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	logger.ConfigureStdLogProcessors("first", func(msg string) (string, bool, bool) {
		if msg == "rewrite-first-match-me" {
			return "[first] " + msg, true, true
		}
		return "", false, false
	})
	logger.ConfigureStdLogProcessors("second", func(msg string) (string, bool, bool) {
		if msg == "rewrite-first-match-me" {
			return "[second] " + msg, true, true
		}
		return "", false, false
	})
	t.Cleanup(func() {
		logger.SetDefault(original)
	})

	log.Println("rewrite-first-match-me")

	records := readAllRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("unexpected record count: %d", len(records))
	}
	if records[0].Level != "DEBUG" || records[0].Message != "rewrite-first-match-me" || records[0].Source.File != "STDLOG" {
		t.Fatalf("unexpected debug record: %+v", records[0])
	}
	record := records[1]
	if record.Message != "[first] rewrite-first-match-me" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if record.Level != "INFO" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Source.File != "STDLOG/first" {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
}

func TestStdLogPrefixIdentityProcessor(t *testing.T) {
	processor := logger.StdLogPrefixIdentityProcessor("deleted key ")

	msg, show, matched := processor("deleted key demo.user")
	if msg != "deleted key demo.user" || !show || !matched {
		t.Fatalf("unexpected processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}

	msg, show, matched = processor("other message")
	if msg != "" || show || matched {
		t.Fatalf("unexpected unmatched processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}
}

func TestStdLogPrefixFilterProcessor(t *testing.T) {
	processor := logger.StdLogPrefixFilterProcessor("deleted key ")

	msg, show, matched := processor("deleted key demo.user")
	if msg != "" || show || !matched {
		t.Fatalf("unexpected processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}

	msg, show, matched = processor("other message")
	if msg != "" || show || matched {
		t.Fatalf("unexpected unmatched processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}
}

func TestStdLogRegexpIdentityProcessor(t *testing.T) {
	processor := logger.StdLogRegexpIdentityProcessor(`deleted key .+`)

	msg, show, matched := processor("deleted key demo.user")
	if msg != "deleted key demo.user" || !show || !matched {
		t.Fatalf("unexpected processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}

	msg, show, matched = processor("other message")
	if msg != "" || show || matched {
		t.Fatalf("unexpected unmatched processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}
}

func TestStdLogRegexpFilterProcessor(t *testing.T) {
	processor := logger.StdLogRegexpFilterProcessor(`deleted key .+`)

	msg, show, matched := processor("deleted key demo.user")
	if msg != "" || show || !matched {
		t.Fatalf("unexpected processor result: msg=%q show=%v matched=%v", msg, show, matched)
	}

	msg, show, matched = processor("other message")
	if msg != "" || show || matched {
		t.Fatalf("unexpected unmatched processor result: msg=%q show=%v matched=%v", msg, show, matched)
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
