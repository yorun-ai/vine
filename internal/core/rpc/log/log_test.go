package log

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
)

func rpcLogTestClientPing() {}

func rpcLogTestServerPing() {}

func rpcLogTestPubPing() {}

func rpcLogTestPubServerPing() {}

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
