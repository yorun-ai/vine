package log

import (
	"encoding/json"
	"errors"
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
	Token  string `json:"token" arg:"1" skel:"sensitive"`
}

type rpcFailingMarshaler struct{}

func (rpcFailingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("secret must not be logged")
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

	span := StartClientInvoke(logger.New("vine:test"), nil, serviceSpec.Methods[0].Info(), "http://127.0.0.1:1/rpc/invoke")
	if !span.muteSuccess {
		t.Fatal("expected muteSuccess span for muteSuccessLog client method")
	}
}

func TestStartClientInvokeRecordsTraceFields(t *testing.T) {
	parent := meta.InitialTrace()
	trace := parent.NewChildTrace()

	span := StartClientInvoke(logger.New("vine:test"), trace, testRpcLogMethodInfo(), "http://127.0.0.1:1/rpc/invoke")

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

	span := StartServerHandle(logger.New("vine:test"), nil, serviceSpec.Methods[0].Info(), nil, nil)
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

func TestRenderRpcPayloadUsesWholeSensitiveMetadata(t *testing.T) {
	method := spec.ConvertSpecToInfoForTest(&spec.ServiceSpec{
		Name:     "SensitiveService",
		SkelName: "rpc.log.sensitive",
		Methods: []*spec.MethodSpec{{
			Name:               "Secret",
			SkelName:           "secret",
			ArgumentsSensitive: true,
			ResultSensitive:    true,
		}},
	}).Methods()[0]

	for name, payload := range map[string]_PayloadValue{
		rpcArgumentsField: renderRpcArguments(method, map[string]string{"value": "secret"}),
		rpcResultField:    renderRpcResult(method, map[string]string{"value": "secret"}),
	} {
		if payload.err != nil || !payload.result.Redacted || payload.result.JSON != `"<redacted>"` {
			t.Fatalf("unexpected %s payload: %#v", name, payload)
		}
	}
}

func TestRenderRpcPayloadReportsTruncation(t *testing.T) {
	result := renderPayload(strings.Repeat("x", 4097), false)
	if result.err != nil || !result.result.Truncated || result.result.Redacted ||
		result.result.JSON != `"<truncated:string bytes=4097>"` {
		t.Fatalf("unexpected truncated Rpc payload: %#v", result)
	}
}

func TestRpcRedactFailureOmitsPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-redact-failure.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})

	StartServerHandle(log, nil, testRpcLogMethodInfo(), nil, nil, rpcFailingMarshaler{})

	records := readRpcLogRecords(t, path)
	if len(records) != 1 || records[0]["rpcArgumentsOmittedReason"] != "redact_failed" {
		t.Fatalf("unexpected lifecycle log: %#v", records)
	}
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
		logger:      logger.New("vine:test"),
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
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})

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
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
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

func TestApplicationPanicUsesErrorLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-application-panic.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
	span := StartServerHandle(log, meta.InitialTrace(), testRpcLogMethodInfo(), nil, nil, &spec.EmptyArguments{})

	var recovered ex.Error
	func() {
		defer func() { recovered = ex.RecoverExecution(recover()) }()
		ex.PanicNew(ex.OperationFailed, "boom")
	}()
	span.FinishServer(recovered, nil)

	records := readRpcLogRecords(t, path)
	finished := records[len(records)-1]
	if finished["level"] != "ERROR" || finished["code"] != string(ex.OperationFailed) || finished["panic"] == "" {
		t.Fatalf("application panic must use Error: %#v", finished)
	}
}

func TestFailureLevelsRemainVisibleWithoutDebug(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-failure-levels.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelInfo, OutputPath: path})
	method := testRpcLogMethodInfo()

	StartServerHandle(log, meta.InitialTrace(), method, nil, nil, &spec.EmptyArguments{}).
		FinishServer(ex.New(ex.OperationFailed, "application"), nil)
	StartServerHandle(log, meta.InitialTrace(), method, nil, nil, &spec.EmptyArguments{}).
		FinishServer(ex.New(ex.InvocationFailed, "system"), nil)
	StartServerHandle(log, meta.InitialTrace(), method, nil, nil, &spec.EmptyArguments{}).
		FinishServer(nil, nil)

	records := readRpcLogRecords(t, path)
	if len(records) != 2 {
		t.Fatalf("Info threshold should emit only failures: %#v", records)
	}
	if records[0]["level"] != "INFO" || records[0]["code"] != string(ex.OperationFailed) {
		t.Fatalf("unexpected application failure: %#v", records[0])
	}
	if records[1]["level"] != "ERROR" || records[1]["code"] != string(ex.InvocationFailed) {
		t.Fatalf("unexpected system failure: %#v", records[1])
	}
}

func TestServerSpanBuildsLifecycleFieldsLazilyWithoutDebug(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-lazy-fields.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelInfo, OutputPath: path})
	method := testRpcLogMethodInfo()
	trace := meta.InitialTrace()
	client := meta.MustNewApp("client", "1.0.0", "123e4567-e89b-12d3-a456-426614174001")
	server := meta.MustNewApp("server", "1.0.0", "123e4567-e89b-12d3-a456-426614174002")

	successSpan := StartServerHandle(log, trace, method, client, server, &spec.EmptyArguments{})
	if successSpan.fieldsInitialized || successSpan.fields != nil {
		t.Fatalf("server lifecycle fields initialized while Debug was disabled: %#v", successSpan.fields)
	}
	successSpan.FinishServer(nil, nil)
	if successSpan.fieldsInitialized || successSpan.fields != nil {
		t.Fatalf("successful server lifecycle fields initialized while Debug stayed disabled: %#v", successSpan.fields)
	}

	failureSpan := StartServerHandle(log, trace, method, client, server, &spec.EmptyArguments{})
	failureSpan.FinishServer(ex.New(ex.OperationFailed, "boom"), nil)
	if !failureSpan.fieldsInitialized {
		t.Fatal("failure did not initialize server lifecycle fields")
	}
	assertSpanField(t, failureSpan, "vrpcId", trace.Id())
	assertSpanField(t, failureSpan, "clientName", client.Name())
	assertSpanField(t, failureSpan, "serverName", server.Name())

	records := readRpcLogRecords(t, path)
	if len(records) != 1 || records[0]["rpcMethod"] != method.Name() || records[0]["clientName"] != client.Name() || records[0]["serverName"] != server.Name() {
		t.Fatalf("lazy failure lifecycle fields were not logged: %#v", records)
	}
}

func TestServerSpanBuildsLifecycleFieldsWhenDebugEnablesBeforeFinish(t *testing.T) {
	t.Cleanup(func() { logger.SetGlobalLevel(logger.LevelInfo) })
	logger.SetGlobalLevel(logger.LevelInfo)
	span := StartServerHandle(logger.New("vine:test"), meta.InitialTrace(), testRpcLogMethodInfo(), nil, nil, &spec.EmptyArguments{})
	if span.fieldsInitialized {
		t.Fatal("server lifecycle fields initialized while Debug was disabled")
	}

	logger.SetGlobalLevel(logger.LevelDebug)
	span.FinishServer(nil, nil)
	if !span.fieldsInitialized {
		t.Fatal("server lifecycle fields were not initialized after Debug was enabled")
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
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})
	arguments := &rpcLifecycleArguments{UserID: "before", Token: "secret"}

	span := StartServerHandle(log, nil, method, nil, nil, arguments)
	arguments.UserID = "after"
	span.FinishServer(ex.New(ex.OperationFailed, "boom"), nil)

	records := readRpcLogRecords(t, path)
	if len(records) != 1 || records[0]["msg"] != "rpc server handle finished" {
		t.Fatalf("muted failure should emit one finished record: %#v", records)
	}
	if records[0]["level"] != "INFO" {
		t.Fatalf("application failure should use Info: %#v", records[0])
	}
	payload, _ := records[0]["rpcArguments"].(string)
	if !strings.Contains(payload, `"userId":"before"`) || strings.Contains(payload, "after") || strings.Contains(payload, "secret") {
		t.Fatalf("muted failure did not use safe start snapshot: %s", payload)
	}
}

func TestMutedFailureMarksArgumentsOmittedWhenDebugWasDisabledAtStart(t *testing.T) {
	t.Cleanup(func() { logger.SetGlobalLevel(logger.LevelInfo) })
	logger.SetGlobalLevel(logger.LevelInfo)
	span := &Span{
		logger:              logger.New("vine:test"),
		finishMsg:           "rpc server handle finished",
		startedAt:           time.Now(),
		muteSuccess:         true,
		debugEnabledAtStart: false,
	}
	logger.SetGlobalLevel(logger.LevelDebug)
	span.FinishServer(ex.New(ex.OperationFailed, "boom"), nil)

	assertSpanField(t, span, "rpcArgumentsOmittedReason", "debug_disabled_at_start")
}

type _CountingMarshaler struct {
	calls *int
}

func (m _CountingMarshaler) MarshalJSON() ([]byte, error) {
	*m.calls++
	return []byte(`"value"`), nil
}

func TestDisabledDebugDoesNotRenderPayload(t *testing.T) {
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
	renderCalls := 0
	value := _CountingMarshaler{calls: &renderCalls}
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatText, Level: logger.LevelInfo})

	span := StartServerHandle(log, nil, method, nil, nil, value)
	span.FinishServer(nil, value)
	if renderCalls != 0 {
		t.Fatalf("payload rendered while Debug was disabled: %d", renderCalls)
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
	path := filepath.Join(t.TempDir(), "rpc-internal-envelope.jsonl")
	log := logger.New("vine:test", logger.WithOption{Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: path})

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
