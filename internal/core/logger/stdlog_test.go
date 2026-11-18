package logger_test

import (
	"encoding/json/v2"
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
	Logger  string       `json:"logger"`
	Message string       `json:"msg"`
	Source  loggedSource `json:"source"`
}

func TestLoggerInfoUsesExternalCallerSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logger.jsonl")
	log := logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	})

	logFromHelper(log)

	record := readLastRecord(t, path)
	if record.Message != "helper-log" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if record.Logger != "vine:test" {
		t.Fatalf("unexpected logger: %q", record.Logger)
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
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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

	customLogger := logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelDebug,
		OutputPath: customPath,
	})
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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
	if records[0].Level != "DEBUG" || records[0].Logger != "vine:stdlog" ||
		records[0].Message != "stdlog-bridge" || records[0].Source.File != "STDLOG" {
		t.Fatalf("unexpected debug record: %+v", records[0])
	}
	record := records[1]
	if record.Message != "stdlog-bridge" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if record.Level != "INFO" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Logger != "vine:stdlog" {
		t.Fatalf("unexpected logger: %q", record.Logger)
	}
	if record.Source.File != "STDLOG" {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
}

func TestStandardLogProcessorDropsMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog-drop.jsonl")
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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
	original := logger.New("vine:default")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
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

func TestStdLogProcessors(t *testing.T) {
	const message = "deleted key demo.user"
	for _, tt := range []struct {
		name        string
		process     logger.StdLogProcessor
		wantMessage string
		wantShow    bool
	}{
		{"prefix identity", logger.StdLogPrefixIdentityProcessor("deleted key "), message, true},
		{"prefix filter", logger.StdLogPrefixFilterProcessor("deleted key "), "", false},
		{"regexp identity", logger.StdLogRegexpIdentityProcessor(`deleted key .+`), message, true},
		{"regexp filter", logger.StdLogRegexpFilterProcessor(`deleted key .+`), "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			msg, show, matched := tt.process(message)
			if msg != tt.wantMessage || show != tt.wantShow || !matched {
				t.Errorf("matched message: got (%q, %v, %v), want (%q, %v, true)", msg, show, matched, tt.wantMessage, tt.wantShow)
			}
			if msg, show, matched := tt.process("other message"); msg != "" || show || matched {
				t.Errorf("unmatched message: got (%q, %v, %v)", msg, show, matched)
			}
		})
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
