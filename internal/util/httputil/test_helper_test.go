package httputil

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"go.yorun.ai/vine/internal/core/logger"
)

func newTestProxyLogger(t *testing.T, name string) (*logger.Logger, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proxy.jsonl")
	log := logger.New(name, logger.WithOption{
		Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path,
	})
	return log, path
}

func readProxyLog(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(body, &record); err != nil {
		t.Fatal(err)
	}
	return record
}
