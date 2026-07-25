package logger

import (
	"errors"
	"strings"
	"testing"
)

type payloadTestValue struct {
	Name          string `json:"name"`
	AccessToken   string `json:"access_token"`
	Authorization string `json:"Authorization"`
	Data          []byte `json:"data"`
}

type payloadPanicMarshaler struct{}

func (payloadPanicMarshaler) MarshalJSON() ([]byte, error) {
	panic("marshal boom")
}

func resetPayloadPoliciesForTest(t *testing.T) {
	t.Helper()
	payloadPolicyMu.Lock()
	previousFrozen := payloadPoliciesFrozen
	previousSnapshot := payloadPolicySnapshot.Load()
	payloadPoliciesFrozen = false
	payloadPolicySnapshot.Store(new(_PayloadPolicySnapshot{
		rpc:     map[_RpcPayloadPolicyKey]PayloadPolicy{},
		event:   map[string]PayloadPolicy{},
		surface: map[PayloadSurface]PayloadPolicy{},
	}))
	payloadPolicyMu.Unlock()
	t.Cleanup(func() {
		payloadPolicyMu.Lock()
		payloadPoliciesFrozen = previousFrozen
		payloadPolicySnapshot.Store(previousSnapshot)
		payloadPolicyMu.Unlock()
	})
}

func TestRenderPayloadRedactsSensitiveKeysAndBinary(t *testing.T) {
	resetPayloadPoliciesForTest(t)
	result := RenderPayload(PayloadDescriptor{Surface: PayloadSurfaceRpcArguments}, payloadTestValue{
		Name:          "Alice",
		AccessToken:   "secret-token",
		Authorization: "Bearer secret",
		Data:          []byte("binary-secret"),
	})

	if result.OmittedReason != "" {
		t.Fatalf("unexpected omission: %s", result.OmittedReason)
	}
	if !result.Redacted {
		t.Fatal("expected redaction marker")
	}
	if strings.Contains(result.JSON, "secret-token") || strings.Contains(result.JSON, "Bearer secret") || strings.Contains(result.JSON, "binary-secret") {
		t.Fatalf("sensitive payload leaked: %s", result.JSON)
	}
	if !strings.Contains(result.JSON, `"access_token":"<redacted>"`) || !strings.Contains(result.JSON, `"data":"<binary:13 bytes>"`) {
		t.Fatalf("unexpected safe projection: %s", result.JSON)
	}
}

func TestRenderPayloadHandlesCycles(t *testing.T) {
	resetPayloadPoliciesForTest(t)
	value := map[string]any{}
	value["self"] = value
	result := RenderPayload(PayloadDescriptor{Surface: PayloadSurfaceEvent}, value)
	if result.OmittedReason != "" || !strings.Contains(result.JSON, "<cycle>") {
		t.Fatalf("unexpected cycle projection: %#v", result)
	}
}

func TestExactUnsafeAndOffPolicies(t *testing.T) {
	resetPayloadPoliciesForTest(t)
	RegisterRpcPayloadPolicy("demo.Service", "get", PayloadSurfaceRpcArguments, PayloadPolicy{Mode: PayloadModeUnsafeFull})
	RegisterEventPayloadPolicy("demo.SecretEvent", PayloadPolicy{Mode: PayloadModeOff})

	rpcResult := RenderPayload(PayloadDescriptor{
		Surface:            PayloadSurfaceRpcArguments,
		RpcServiceSkelName: "demo.Service",
		RpcMethodSkelName:  "get",
	}, map[string]string{"token": "visible-for-explicit-unsafe"})
	if !strings.Contains(rpcResult.JSON, "visible-for-explicit-unsafe") || rpcResult.Redacted {
		t.Fatalf("unexpected UNSAFE_FULL projection: %#v", rpcResult)
	}
	eventResult := RenderPayload(PayloadDescriptor{
		Surface:       PayloadSurfaceEvent,
		EventSkelName: "demo.SecretEvent",
	}, map[string]string{"name": "hidden"})
	if eventResult.JSON != "" || eventResult.OmittedReason != "policy_off" {
		t.Fatalf("unexpected OFF projection: %#v", eventResult)
	}
}

func TestPayloadFailuresAreIsolated(t *testing.T) {
	resetPayloadPoliciesForTest(t)
	RegisterEventPayloadPolicy("demo.Event", PayloadPolicy{
		Sanitizer: func(PayloadDescriptor, any) (any, error) {
			return nil, errors.New("cannot sanitize")
		},
	})
	result := RenderPayload(PayloadDescriptor{Surface: PayloadSurfaceEvent, EventSkelName: "demo.Event"}, "value")
	if result.OmittedReason != "serialization_failed" {
		t.Fatalf("unexpected sanitizer failure result: %#v", result)
	}

	result = RenderPayload(PayloadDescriptor{Surface: PayloadSurfaceRpcResult}, payloadPanicMarshaler{})
	if result.OmittedReason != "serialization_failed" {
		t.Fatalf("unexpected marshal panic result: %#v", result)
	}
}
