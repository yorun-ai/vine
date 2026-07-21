package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type _LoggedSource struct {
	File string `json:"file"`
}

type _LoggedRecord struct {
	Message string        `json:"msg"`
	Source  _LoggedSource `json:"source"`
}

func TestFacadeInfoUsesExternalCallerSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facade-logger.jsonl")
	log := NewLogger(&Option{
		Mode:       ModeJSON,
		Level:      LevelDebug,
		OutputPath: path,
	})

	logFromFacadeHelper(log)

	record := readFacadeLastRecord(t, path)
	if record.Message != "facade-helper-log" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if !strings.HasSuffix(record.Source.File, "api_test.go") {
		t.Fatalf("unexpected source file: %q", record.Source.File)
	}
	if strings.Contains(record.Source.File, "core/logger/api.go") {
		t.Fatalf("unexpected facade wrapper source file: %q", record.Source.File)
	}
}

func logFromFacadeHelper(log *Logger) {
	log.Info("facade-helper-log")
}

func readFacadeLastRecord(t *testing.T, path string) _LoggedRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected at least one log line")
	}

	var record _LoggedRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &record); err != nil {
		t.Fatalf("unmarshal log record: %v", err)
	}
	return record
}
