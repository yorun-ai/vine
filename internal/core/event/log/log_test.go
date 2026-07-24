package log

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
)

type eventLogTestInfo struct{}

func (eventLogTestInfo) Name() string                        { return "Created" }
func (eventLogTestInfo) SkelName() string                    { return "test.event.Created" }
func (eventLogTestInfo) Hash() string                        { return "" }
func (eventLogTestInfo) PayloadType() reflect.Type           { return reflect.TypeFor[map[string]any]() }
func (eventLogTestInfo) EmitterMethodName() string           { return "EmitCreated" }
func (eventLogTestInfo) EmitterType() reflect.Type           { return nil }
func (eventLogTestInfo) EmitterCtor() any                    { return nil }
func (eventLogTestInfo) ListenerMethodName() string          { return "OnCreated" }
func (eventLogTestInfo) ListenerType() reflect.Type          { return nil }
func (eventLogTestInfo) DefaultListenerType() reflect.Type   { return nil }
func (eventLogTestInfo) ERListenerType() reflect.Type        { return nil }
func (eventLogTestInfo) WrapperERListenerCtor() any          { return nil }
func (eventLogTestInfo) DefaultERListenerType() reflect.Type { return nil }
func (eventLogTestInfo) NewEvent() any                       { return new(map[string]any) }

var _ spec.EventInfo = eventLogTestInfo{}

func TestListenerStartedLogsSafePayloadOnlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartListenerHandle(log, meta.InitialTrace(), eventLogTestInfo{}, nil, nil, map[string]any{
		"userId": "u-1",
		"cookie": "private-cookie",
	})
	span.Finish(nil)

	records := readEventLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("expected started and finished, got %#v", records)
	}
	started := records[0]
	payload, _ := started["eventPayload"].(string)
	if started["level"] != "DEBUG" || strings.Contains(payload, "private-cookie") || !strings.Contains(payload, `"cookie":"<redacted>"`) {
		t.Fatalf("unexpected started record: %#v", started)
	}
	finished := records[1]
	if finished["level"] != "DEBUG" || finished["code"] != "OK" {
		t.Fatalf("unexpected finished record: %#v", finished)
	}
	if _, exists := finished["eventPayload"]; exists {
		t.Fatalf("finished must not repeat Event payload: %#v", finished)
	}
}

func TestListenerPanicUsesErrorAndMainFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-panic.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartListenerHandle(log, meta.InitialTrace(), eventLogTestInfo{}, nil, nil, map[string]any{})

	var recovered ex.Error
	func() {
		defer func() { recovered = ex.RecoverExecution(recover()) }()
		panic("boom")
	}()
	span.Finish(recovered)

	records := readEventLogRecords(t, path)
	finished := records[len(records)-1]
	if finished["level"] != "ERROR" || finished["panic"] != "boom" || finished["stack"] == "" {
		t.Fatalf("unexpected panic record: %#v", finished)
	}
	if finished["eventSkel"] != "test.event.Created" || finished["eventEmitterMethod"] != "EmitCreated" || finished["eventListenerMethod"] != "OnCreated" {
		t.Fatalf("main Event fields are missing: %#v", finished)
	}
	for _, key := range []string{"eventSkelName", "listenerMethod"} {
		if _, exists := finished[key]; exists {
			t.Fatalf("unexpected duplicate Event field %s in %#v", key, finished)
		}
	}
}

func TestListenerApplicationFailureUsesInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-application-error.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartListenerHandle(log, meta.InitialTrace(), eventLogTestInfo{}, nil, nil, map[string]any{})

	span.Finish(ex.New(ex.OperationFailed, "boom"))

	records := readEventLogRecords(t, path)
	finished := records[len(records)-1]
	if finished["level"] != "INFO" || finished["code"] != string(ex.OperationFailed) {
		t.Fatalf("unexpected application failure record: %#v", finished)
	}
}

func readEventLogRecords(t *testing.T, path string) []map[string]any {
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
