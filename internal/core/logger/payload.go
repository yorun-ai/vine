package logger

import (
	"sync"
	"sync/atomic"

	"go.yorun.ai/vine/internal/core/redact"
	"go.yorun.ai/vine/util/vpre"
)

type PayloadSurface string

const (
	PayloadSurfaceRpcArguments PayloadSurface = "rpcArguments"
	PayloadSurfaceRpcResult    PayloadSurface = "rpcResult"
	PayloadSurfaceEvent        PayloadSurface = "eventPayload"
)

type PayloadMode string

const (
	PayloadModeSafe       PayloadMode = "SAFE"
	PayloadModeOff        PayloadMode = "OFF"
	PayloadModeUnsafeFull PayloadMode = "UNSAFE_FULL"
)

type PayloadDescriptor struct {
	// Surface identifies the log field being rendered.
	Surface PayloadSurface
	// Sensitive marks the entire payload as sensitive.
	Sensitive bool
	// RpcServiceSkelName and RpcMethodSkelName identify an exact Rpc selector.
	RpcServiceSkelName string
	RpcMethodSkelName  string
	// EventSkelName identifies an exact Event selector.
	EventSkelName string
}

type PayloadSanitizer func(PayloadDescriptor, any) (any, error)

type PayloadPolicy struct {
	// Mode defaults to SAFE when empty.
	Mode PayloadMode
	// Sanitizer optionally creates a domain-specific projection before built-in redaction.
	Sanitizer PayloadSanitizer
}

type PayloadValue struct {
	JSON          string
	Redacted      bool
	OmittedReason string
}

type _RpcPayloadPolicyKey struct {
	service string
	method  string
	surface PayloadSurface
}

var payloadPolicyMu sync.RWMutex
var payloadPoliciesFrozen bool

type _PayloadPolicySnapshot struct {
	rpc     map[_RpcPayloadPolicyKey]PayloadPolicy
	event   map[string]PayloadPolicy
	surface map[PayloadSurface]PayloadPolicy
}

var payloadPolicySnapshot atomic.Pointer[_PayloadPolicySnapshot]

func init() {
	payloadPolicySnapshot.Store(new(_PayloadPolicySnapshot{
		rpc:     map[_RpcPayloadPolicyKey]PayloadPolicy{},
		event:   map[string]PayloadPolicy{},
		surface: map[PayloadSurface]PayloadPolicy{},
	}))
}

func RegisterRpcPayloadPolicy(serviceSkelName string, methodSkelName string, surface PayloadSurface, policy PayloadPolicy) {
	validatePayloadSurface(surface)
	vpre.Check(surface == PayloadSurfaceRpcArguments || surface == PayloadSurfaceRpcResult,
		"Rpc payload policy requires an Rpc payload surface")
	vpre.Check(serviceSkelName != "", "rpc payload policy service name cannot be empty")
	vpre.Check(methodSkelName != "", "rpc payload policy method name cannot be empty")
	validatePayloadPolicy(policy, true)
	payloadPolicyMu.Lock()
	defer payloadPolicyMu.Unlock()
	vpre.Check(!payloadPoliciesFrozen, "payload policy registry is frozen")
	next := clonePayloadPolicySnapshot(payloadPolicySnapshot.Load())
	next.rpc[_RpcPayloadPolicyKey{service: serviceSkelName, method: methodSkelName, surface: surface}] = normalizePayloadPolicy(policy)
	payloadPolicySnapshot.Store(next)
}

func RegisterEventPayloadPolicy(eventSkelName string, policy PayloadPolicy) {
	vpre.Check(eventSkelName != "", "event payload policy event name cannot be empty")
	validatePayloadPolicy(policy, true)
	payloadPolicyMu.Lock()
	defer payloadPolicyMu.Unlock()
	vpre.Check(!payloadPoliciesFrozen, "payload policy registry is frozen")
	next := clonePayloadPolicySnapshot(payloadPolicySnapshot.Load())
	next.event[eventSkelName] = normalizePayloadPolicy(policy)
	payloadPolicySnapshot.Store(next)
}

func RegisterPayloadSurfacePolicy(surface PayloadSurface, policy PayloadPolicy) {
	validatePayloadSurface(surface)
	validatePayloadPolicy(policy, false)
	payloadPolicyMu.Lock()
	defer payloadPolicyMu.Unlock()
	vpre.Check(!payloadPoliciesFrozen, "payload policy registry is frozen")
	next := clonePayloadPolicySnapshot(payloadPolicySnapshot.Load())
	next.surface[surface] = normalizePayloadPolicy(policy)
	payloadPolicySnapshot.Store(next)
}

func FreezePayloadPolicies() {
	payloadPolicyMu.Lock()
	payloadPoliciesFrozen = true
	payloadPolicyMu.Unlock()
}

func RenderPayload(descriptor PayloadDescriptor, value any) (result PayloadValue) {
	policy := resolvePayloadPolicy(descriptor)
	if policy.Mode == PayloadModeOff {
		return PayloadValue{OmittedReason: "policy_off"}
	}

	defer func() {
		if recover() != nil {
			result = PayloadValue{OmittedReason: "serialization_failed"}
		}
	}()

	if policy.Sanitizer != nil {
		var err error
		value, err = policy.Sanitizer(descriptor, value)
		if err != nil {
			return PayloadValue{OmittedReason: "serialization_failed"}
		}
	}

	rendered, err := redact.Render(value, redact.Option{
		RevealSensitive: policy.Mode == PayloadModeUnsafeFull,
		Sensitive:       descriptor.Sensitive,
	})
	if err != nil {
		return PayloadValue{OmittedReason: "serialization_failed"}
	}
	return PayloadValue{
		JSON:     rendered.JSON,
		Redacted: rendered.Redacted,
	}
}

func resolvePayloadPolicy(descriptor PayloadDescriptor) PayloadPolicy {
	validatePayloadSurface(descriptor.Surface)
	snapshot := payloadPolicySnapshot.Load()
	if descriptor.RpcServiceSkelName != "" && descriptor.RpcMethodSkelName != "" {
		if policy, ok := snapshot.rpc[_RpcPayloadPolicyKey{
			service: descriptor.RpcServiceSkelName,
			method:  descriptor.RpcMethodSkelName,
			surface: descriptor.Surface,
		}]; ok {
			return policy
		}
	}
	if descriptor.EventSkelName != "" {
		if policy, ok := snapshot.event[descriptor.EventSkelName]; ok {
			return policy
		}
	}
	if policy, ok := snapshot.surface[descriptor.Surface]; ok {
		return policy
	}
	return PayloadPolicy{Mode: PayloadModeSafe}
}

func clonePayloadPolicySnapshot(source *_PayloadPolicySnapshot) *_PayloadPolicySnapshot {
	next := new(_PayloadPolicySnapshot{
		rpc:     make(map[_RpcPayloadPolicyKey]PayloadPolicy, len(source.rpc)),
		event:   make(map[string]PayloadPolicy, len(source.event)),
		surface: make(map[PayloadSurface]PayloadPolicy, len(source.surface)),
	})
	for key, policy := range source.rpc {
		next.rpc[key] = policy
	}
	for key, policy := range source.event {
		next.event[key] = policy
	}
	for key, policy := range source.surface {
		next.surface[key] = policy
	}
	return next
}

func normalizePayloadPolicy(policy PayloadPolicy) PayloadPolicy {
	if policy.Mode == "" {
		policy.Mode = PayloadModeSafe
	}
	return policy
}

func validatePayloadPolicy(policy PayloadPolicy, exact bool) {
	mode := normalizePayloadPolicy(policy).Mode
	vpre.Check(mode == PayloadModeSafe || mode == PayloadModeOff || mode == PayloadModeUnsafeFull,
		"%+v is not a valid payload mode", mode)
	vpre.Check(exact || mode != PayloadModeUnsafeFull, "UNSAFE_FULL requires an exact Rpc method or Event selector")
}

func validatePayloadSurface(surface PayloadSurface) {
	vpre.Check(surface == PayloadSurfaceRpcArguments ||
		surface == PayloadSurfaceRpcResult ||
		surface == PayloadSurfaceEvent, "%+v is not a valid payload surface", surface)
}
