package log

import (
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/task/spec"
)

func TestRunnerFinishedUsesDebugAndOmitsOKError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
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

type taskLogArguments struct {
	Value string `json:"value"`
}

type taskFailingMarshaler struct{}

func (taskFailingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("secret must not be logged")
}

func TestRunnerStartedLogsRedactedArgumentsOnlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-arguments.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
	taskInfo := spec.ConvertSpecToInfoForTest(new(spec.TaskSpec{
		Name:     "SecretTask",
		SkelName: "test.task.Secret",
		Triggers: []*spec.TriggerSpec{{
			Name:               "Run",
			SkelName:           "run",
			LauncherMethodName: "LaunchRun",
			RunnerMethodName:   "Run",
			ArgumentsType:      reflect.TypeFor[taskLogArguments](),
			ArgumentsSensitive: true,
		}},
	}))
	span := StartRunnerHandle(
		log,
		meta.InitialTrace(),
		taskInfo.Triggers()[0],
		nil,
		nil,
		&taskLogArguments{Value: "secret"},
	)
	span.Finish(nil)

	records := readTaskLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("expected started and finished, got %#v", records)
	}
	if records[0]["taskArguments"] != `"<redacted>"` ||
		records[0]["taskArgumentsRedacted"] != true {
		t.Fatalf("unexpected started arguments: %#v", records[0])
	}
	if _, exists := records[1]["taskArguments"]; exists {
		t.Fatalf("finished must not repeat Task arguments: %#v", records[1])
	}
}

func TestTaskRedactFailureOmitsPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-redact-failure.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})

	StartRunnerHandle(log, nil, nil, nil, nil, taskFailingMarshaler{})

	records := readTaskLogRecords(t, path)
	if len(records) != 1 || records[0]["taskArgumentsOmittedReason"] != "redact_failed" {
		t.Fatalf("unexpected lifecycle log: %#v", records)
	}
}

func TestRunnerFailureIncludesSourceStack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-error.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
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
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	return records
}
