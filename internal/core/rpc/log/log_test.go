package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

type rpcLifecycleArguments struct {
	UserID string `json:"userId" arg:"0"`
	Token  string `json:"token" arg:"1"`
}

func rpcLogTestClientPing() {}

func rpcLogTestServerPing() {}

func rpcLogTestPubPing() {}

func rpcLogTestPubServerPing() {}

func rpcLogTestMutedFailure(string, string) {}

func resetMuteForTest(t *testing.T) {
	t.Helper()

	prevMuteSuccessLogMethodKeys := muteSuccessLogMethodKeys
	muteSuccessLogMethodKeys = map[_MuteSuccessLogMethodKey]struct{}{}
	t.Cleanup(func() {
		muteSuccessLogMethodKeys = prevMuteSuccessLogMethodKeys
	})
}

func TestStartClientInvokeMutesSuccessLogWhenMethodMuteSuccessLog(t *testing.T) {
	resetMuteForTest(t)

	serviceSpec := &spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeClient,
		Name:     "RpcLogTestService",
		SkelName: "rpc.log.test.client",
		Methods: []*spec.MethodSpec{{
			Name:        "Ping",
			SkelName:    "ping",
			MethodFuncs: []any{rpcLogTestClientPing},
		}},
	}
	spec.Register(serviceSpec)
	MuteSuccessLog(rpcLogTestClientPing)

	span := StartClientInvoke(logger.NewLogger(logger.GlobalOption()), nil, serviceSpec.Methods[0].Info(), "http://127.0.0.1:1/rpc/invoke")
	if !span.muteSuccess {
		t.Fatal("expected muteSuccess span for muteSuccessLog client method")
	}
}

func TestStartClientInvokeRecordsTraceFields(t *testing.T) {
	parent := meta.InitialTrace()
	trace := parent.NewChildTrace()

	span := StartClientInvoke(logger.NewLogger(logger.GlobalOption()), trace, testRpcLogMethodInfo(), "http://127.0.0.1:1/rpc/invoke")

	assertSpanField(t, span, "vrpcId", trace.Id())
	assertSpanField(t, span, "vrpcSpan", trace.Span())
	assertSpanField(t, span, "vrpcParentSpan", parent.Span())
}

func TestStartServerHandleMutesSuccessLogWhenMethodMuteSuccessLog(t *testing.T) {
	resetMuteForTest(t)

	serviceSpec := &spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeServer,
		Name:     "RpcLogTestService",
		SkelName: "rpc.log.test.server",
		Methods: []*spec.MethodSpec{{
			Name:          "Ping",
			SkelName:      "ping",
			ArgumentsType: reflect.TypeFor[spec.EmptyArguments](),
			MethodFuncs:   []any{rpcLogTestServerPing},
		}},
	}
	spec.Register(serviceSpec)
	MuteSuccessLog(rpcLogTestServerPing)

	span := StartServerHandle(logger.NewLogger(logger.GlobalOption()), nil, serviceSpec.Methods[0].Info(), nil, nil)
	if !span.muteSuccess {
		t.Fatal("expected muteSuccess span for muteSuccessLog server method")
	}
}

func testRpcLogMethodInfo() spec.MethodInfo {
	return spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "RpcLogTraceTestService",
		SkelName: "rpc.log.trace.test",
		Methods: []*spec.MethodSpec{{
			Name:     "Ping",
			SkelName: "ping",
		}},
	}).Methods()[0]
}

func assertSpanField(t *testing.T, span *Span, key string, want any) {
	t.Helper()

	for i := 0; i < len(span.fields)-1; i += 2 {
		if span.fields[i] == key {
			if span.fields[i+1] != want {
				t.Fatalf("%s = %#v, want %#v", key, span.fields[i+1], want)
			}
			return
		}
	}
	t.Fatalf("field %s not found in %#v", key, span.fields)
}

func TestMuteSuccessLogMatchesFutureServerMethodBySkelName(t *testing.T) {
	resetMuteForTest(t)

	pubServiceSpec := &spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeClient,
		Name:     "RpcLogPubTestService",
		SkelName: "rpc.log.test.pub",
		Methods: []*spec.MethodSpec{{
			Name:        "Ping",
			SkelName:    "ping",
			MethodFuncs: []any{rpcLogTestPubPing},
		}},
	}
	spec.Register(pubServiceSpec)

	MuteSuccessLog(rpcLogTestPubPing)

	serverServiceSpec := &spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeServer,
		Name:     "RpcLogPubTestService",
		SkelName: "rpc.log.test.pub",
		Methods: []*spec.MethodSpec{{
			Name:          "Ping",
			SkelName:      "ping",
			ArgumentsType: reflect.TypeFor[spec.EmptyArguments](),
			MethodFuncs:   []any{rpcLogTestPubServerPing},
		}},
	}
	spec.Register(serverServiceSpec)

	methodInfo, ok := spec.GetMethodInfo(serverServiceSpec.SkelName, "ping")
	if !ok {
		t.Fatal("expected server method info")
	}
	if !IsSuccessLogMuted(methodInfo) {
		t.Fatal("expected future server method success log to be muted")
	}
}

func TestMuteSuccessLogRejectsUnknownMethod(t *testing.T) {
	resetMuteForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(fmt.Sprint(recovered), "unknown rpc service method") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	MuteSuccessLog(func() {})
}

func TestMuteSuccessSpanStillLogsError(t *testing.T) {
	span := &Span{
		logger:      logger.NewLogger(logger.GlobalOption()),
		muteSuccess: true,
	}

	span.Finish(ex.New(ex.InvocationFailed, "boom"))
}

func TestServerLifecycleLogsSafePayloadAndDebugFinished(t *testing.T) {
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "LifecycleService",
		SkelName: "test.lifecycle.Service",
		Methods: []*spec.MethodSpec{{
			Name:          "Get",
			SkelName:      "get",
			ArgumentsType: reflect.TypeFor[rpcLifecycleArguments](),
			ResultType:    reflect.TypeFor[map[string]string](),
		}},
	}).Methods()[0]
	path := filepath.Join(t.TempDir(), "rpc.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})

	span := StartServerHandle(log, meta.InitialTrace(), method, nil, nil, &rpcLifecycleArguments{
		UserID: "u-1",
		Token:  "secret-token",
	})
	span.FinishServer(nil, map[string]string{"name": "Alice"})

	records := readRpcLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("expected started and finished records, got %#v", records)
	}
	if records[0]["level"] != "DEBUG" || records[0]["msg"] != "rpc server handle started" {
		t.Fatalf("unexpected started record: %#v", records[0])
	}
	arguments, _ := records[0]["rpcArguments"].(string)
	if strings.Contains(arguments, "secret-token") || !strings.Contains(arguments, `"token":"<redacted>"`) {
		t.Fatalf("unexpected Rpc arguments: %s", arguments)
	}
	if records[1]["level"] != "DEBUG" || records[1]["code"] != "OK" {
		t.Fatalf("unexpected finished record: %#v", records[1])
	}
	if _, exists := records[1]["error"]; exists {
		t.Fatalf("successful finished record must not contain error: %#v", records[1])
	}
	if result, _ := records[1]["rpcResult"].(string); !strings.Contains(result, `"name":"Alice"`) {
		t.Fatalf("unexpected Rpc result: %s", result)
	}
}

func TestPanicDiagnosticIsMergedIntoSingleFailureFinished(t *testing.T) {
	method := testRpcLogMethodInfo()
	path := filepath.Join(t.TempDir(), "rpc-panic.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartServerHandle(log, meta.InitialTrace(), method, nil, nil, &spec.EmptyArguments{})

	var recovered ex.Error
	func() {
		defer func() { recovered = ex.RecoverExecution(recover()) }()
		panic("boom")
	}()
	span.FinishServer(recovered, nil)

	records := readRpcLogRecords(t, path)
	finished := 0
	for _, record := range records {
		if record["msg"] == "rpc server handle finished" {
			finished++
			if record["level"] != "ERROR" || record["panic"] != "boom" || record["stack"] == "" {
				t.Fatalf("unexpected panic finished record: %#v", record)
			}
		}
		if strings.Contains(fmt.Sprint(record["msg"]), "recovered panic") {
			t.Fatalf("unexpected duplicate recovery record: %#v", record)
		}
	}
	if finished != 1 {
		t.Fatalf("expected one terminal finished record, got %d in %#v", finished, records)
	}
}

func TestMutedMethodLogsOnlyFailureFinishedWithStartSnapshot(t *testing.T) {
	resetMuteForTest(t)
	serviceSpec := &spec.ServiceSpec{
		Type:     spec.ServiceSpecTypeServer,
		Name:     "MutedFailureService",
		SkelName: "rpc.log.test.muted.failure",
		Methods: []*spec.MethodSpec{{
			Name:          "Run",
			SkelName:      "run",
			ArgumentsType: reflect.TypeFor[rpcLifecycleArguments](),
			MethodFuncs:   []any{rpcLogTestMutedFailure},
		}},
	}
	spec.Register(serviceSpec)
	MuteSuccessLog(rpcLogTestMutedFailure)
	method := serviceSpec.Methods[0].Info()
	path := filepath.Join(t.TempDir(), "rpc-muted.jsonl")
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path})
	arguments := &rpcLifecycleArguments{UserID: "before", Token: "secret"}

	span := StartServerHandle(log, nil, method, nil, nil, arguments)
	arguments.UserID = "after"
	span.FinishServer(ex.New(ex.OperationFailed, "boom"), nil)

	records := readRpcLogRecords(t, path)
	if len(records) != 1 || records[0]["msg"] != "rpc server handle finished" {
		t.Fatalf("muted failure should emit one finished record: %#v", records)
	}
	payload, _ := records[0]["rpcArguments"].(string)
	if !strings.Contains(payload, `"userId":"before"`) || strings.Contains(payload, "after") || strings.Contains(payload, "secret") {
		t.Fatalf("muted failure did not use safe start snapshot: %s", payload)
	}
}

func TestMutedFailureMarksArgumentsOmittedWhenDebugWasDisabledAtStart(t *testing.T) {
	previousLevel := logger.GlobalOption().Level
	t.Cleanup(func() { logger.SetGlobalLevel(previousLevel) })
	logger.SetGlobalLevel(logger.LevelInfo)
	span := &Span{
		logger:              logger.NewGlobalLogger(),
		finishMsg:           "rpc server handle finished",
		startedAt:           time.Now(),
		muteSuccess:         true,
		debugEnabledAtStart: false,
	}
	logger.SetGlobalLevel(logger.LevelDebug)
	span.FinishServer(ex.New(ex.OperationFailed, "boom"), nil)

	assertSpanField(t, span, "rpcArgumentsOmittedReason", "debug_disabled_at_start")
}

func TestDisabledDebugDoesNotInvokePayloadSanitizer(t *testing.T) {
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "LazyPayloadService",
		SkelName: "rpc.log.test.lazy.payload",
		Methods: []*spec.MethodSpec{{
			Name:          "Get",
			SkelName:      "get",
			ArgumentsType: reflect.TypeFor[rpcLifecycleArguments](),
			ResultType:    reflect.TypeFor[string](),
		}},
	}).Methods()[0]
	sanitizerCalls := 0
	logger.RegisterRpcPayloadPolicy(method.Service().SkelName(), method.SkelName(), logger.PayloadSurfaceRpcArguments, logger.PayloadPolicy{
		Sanitizer: func(logger.PayloadDescriptor, any) (any, error) {
			sanitizerCalls++
			return map[string]string{"value": "safe"}, nil
		},
	})
	logger.RegisterRpcPayloadPolicy(method.Service().SkelName(), method.SkelName(), logger.PayloadSurfaceRpcResult, logger.PayloadPolicy{
		Sanitizer: func(logger.PayloadDescriptor, any) (any, error) {
			sanitizerCalls++
			return "safe", nil
		},
	})
	log := logger.NewLogger(&logger.Option{Mode: logger.ModeText, Level: logger.LevelInfo})

	span := StartServerHandle(log, nil, method, nil, nil, &rpcLifecycleArguments{})
	span.FinishServer(nil, "result")
	if sanitizerCalls != 0 {
		t.Fatalf("payload sanitizer ran while Debug was disabled: %d", sanitizerCalls)
	}
}

func TestInternalTaskEventTransportNeverLogsRpcPayload(t *testing.T) {
	method := spec.ConvertSpecToInfoForTest(new(spec.ServiceSpec{
		Name:     "EventService",
		SkelName: "vine.app.EventService",
		Methods: []*spec.MethodSpec{{
			Name:          "OnEvent",
			SkelName:      "onEvent",
			ArgumentsType: reflect.TypeFor[rpcLifecycleArguments](),
			ResultType:    reflect.TypeFor[string](),
		}},
	})).Methods()[0]
	logger.RegisterRpcPayloadPolicy(method.Service().SkelName(), method.SkelName(), logger.PayloadSurfaceRpcArguments,
		logger.PayloadPolicy{Mode: logger.PayloadModeUnsafeFull})
	logger.RegisterRpcPayloadPolicy(method.Service().SkelName(), method.SkelName(), logger.PayloadSurfaceRpcResult,
		logger.PayloadPolicy{Mode: logger.PayloadModeUnsafeFull})
	path := filepath.Join(t.TempDir(), "rpc-internal-envelope.jsonl")
	log := logger.NewLogger(new(logger.Option{Mode: logger.ModeJSON, Level: logger.LevelDebug, OutputPath: path}))

	span := StartServerHandle(log, nil, method, nil, nil, new(rpcLifecycleArguments{Token: "secret"}))
	span.FinishServer(nil, "secret-result")

	records := readRpcLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("internal transport should keep lifecycle logs: %#v", records)
	}
	for _, record := range records {
		if _, exists := record["rpcArguments"]; exists {
			t.Fatalf("internal transport arguments leaked: %#v", record)
		}
		if _, exists := record["rpcResult"]; exists {
			t.Fatalf("internal transport result leaked: %#v", record)
		}
	}
}

func readRpcLogRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log record: %v", err)
		}
		records = append(records, record)
	}
	return records
}
