package log

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/task/spec"
)

func TestRunnerFinishedUsesDebugAndOmitsOKError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartRunnerHandle(log, meta.InitialTrace(), nil, nil, nil)
	span.Finish(ex.NewOK())

	records := readTaskLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("expected started and finished, got %#v", records)
	}
	finished := records[1]
	if finished["msg"] != "task runner handle finished" || finished["level"] != "DEBUG" || finished["code"] != "OK" {
		t.Fatalf("unexpected finished record: %#v", finished)
	}
	if _, exists := finished["error"]; exists {
		t.Fatalf("OK finished record must not contain error: %#v", finished)
	}
	if _, exists := finished["duration"]; !exists {
		t.Fatalf("finished record must contain duration: %#v", finished)
	}
}

func TestRunnerFailureIncludesSourceStack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-error.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	taskInfo := spec.ConvertSpecToInfoForTest(new(spec.TaskSpec{
		Name:     "RebuildIndex",
		SkelName: "test.task.RebuildIndex",
		Triggers: []*spec.TriggerSpec{{
			Name:               "Nightly",
			SkelName:           "nightly",
			LauncherMethodName: "LaunchNightly",
			RunnerMethodName:   "RunNightly",
		}},
	}))
	span := StartRunnerHandle(log, meta.InitialTrace(), taskInfo.Triggers()[0], nil, nil)
	span.Finish(ex.New(ex.OperationFailed, "boom"))

	records := readTaskLogRecords(t, path)
	finished := records[len(records)-1]
	if finished["level"] != "INFO" || finished["code"] != string(ex.OperationFailed) || finished["error"] == "" {
		t.Fatalf("unexpected failure record: %#v", finished)
	}
	if stack, _ := finished["stack"].(string); !strings.Contains(stack, "TestRunnerFailureIncludesSourceStack") {
		t.Fatalf("unexpected error stack: %s", stack)
	}
	for key, want := range map[string]any{
		"taskSkel":           "test.task.RebuildIndex",
		"taskTriggerName":    "Nightly",
		"taskTriggerSkel":    "nightly",
		"taskLauncherMethod": "LaunchNightly",
		"taskRunnerMethod":   "RunNightly",
	} {
		if finished[key] != want {
			t.Fatalf("%s = %#v, want %#v in %#v", key, finished[key], want, finished)
		}
	}
	for _, key := range []string{"taskSkelName", "triggerName", "triggerSkelName", "runnerMethod"} {
		if _, exists := finished[key]; exists {
			t.Fatalf("unexpected duplicate Task field %s in %#v", key, finished)
		}
	}
}

func readTaskLogRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read logs: %v", err)
	}
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	return records
}
