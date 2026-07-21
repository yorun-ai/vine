package logger_test

import (
	"log"
	"path/filepath"
	"testing"

	logger "go.yorun.ai/vine/internal/core/logger"
)

func TestStandardLogWritesThroughDefaultLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog.jsonl")
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
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
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
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
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
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
	original := logger.NewLogger(logger.GlobalOption())
	logger.SetDefault(logger.NewLogger(&logger.Option{
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
