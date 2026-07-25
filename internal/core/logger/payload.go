package logger

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

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

	state := new(_PayloadProjectionState{
		mode:     policy.Mode,
		visiting: map[_PayloadVisit]struct{}{},
	})
	projected, err := state.project(reflect.ValueOf(value), "")
	if err != nil {
		return PayloadValue{OmittedReason: "serialization_failed"}
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(projected); err != nil {
		return PayloadValue{OmittedReason: "serialization_failed"}
	}
	return PayloadValue{
		JSON:     strings.TrimSuffix(encoded.String(), "\n"),
		Redacted: state.redacted,
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

type _PayloadVisit struct {
	kind reflect.Kind
	ptr  uintptr
}

type _PayloadProjectionState struct {
	mode     PayloadMode
	visiting map[_PayloadVisit]struct{}
	redacted bool
}

func (s *_PayloadProjectionState) project(value reflect.Value, key string) (any, error) {
	if key != "" && s.mode == PayloadModeSafe && isSensitivePayloadKey(key) {
		s.redacted = true
		return "<redacted>", nil
	}
	if !value.IsValid() {
		return nil, nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}
		if value.Kind() == reflect.Pointer {
			visit := _PayloadVisit{kind: value.Kind(), ptr: value.Pointer()}
			if _, exists := s.visiting[visit]; exists {
				return "<cycle>", nil
			}
			s.visiting[visit] = struct{}{}
			defer delete(s.visiting, visit)
		}
		value = value.Elem()
	}

	if isBinaryPayload(value) {
		return fmt.Sprintf("<binary:%d bytes>", value.Len()), nil
	}
	if marshaled, ok, err := marshalPayloadValue(value); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return s.project(reflect.ValueOf(marshaled), key)
	}

	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.String:
		return strings.ToValidUTF8(value.String(), "�"), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), nil
	case reflect.Float32, reflect.Float64:
		floatValue := value.Float()
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return "<non-finite-number>", nil
		}
		return floatValue, nil
	case reflect.Complex64, reflect.Complex128:
		return fmt.Sprintf("<complex:%v>", value.Complex()), nil
	case reflect.Struct:
		return s.projectStruct(value)
	case reflect.Map:
		return s.projectMap(value)
	case reflect.Slice, reflect.Array:
		return s.projectList(value)
	case reflect.Invalid:
		return nil, nil
	default:
		return "<" + value.Type().String() + ">", nil
	}
}

func (s *_PayloadProjectionState) projectStruct(value reflect.Value) (map[string]any, error) {
	result := map[string]any{}
	for index := range value.NumField() {
		fieldInfo := value.Type().Field(index)
		if fieldInfo.PkgPath != "" {
			continue
		}
		name, embedded, skip := payloadFieldName(fieldInfo)
		if skip {
			continue
		}
		fieldValue := value.Field(index)
		if embedded {
			projected, err := s.project(fieldValue, "")
			if err != nil {
				return nil, err
			}
			if object, ok := projected.(map[string]any); ok {
				for embeddedName, embeddedValue := range object {
					result[embeddedName] = embeddedValue
				}
			}
			continue
		}
		projected, err := s.project(fieldValue, name)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_PayloadProjectionState) projectMap(value reflect.Value) (map[string]any, error) {
	if value.IsNil() {
		return nil, nil
	}
	visit := _PayloadVisit{kind: value.Kind(), ptr: value.Pointer()}
	if _, exists := s.visiting[visit]; exists {
		return map[string]any{"_value": "<cycle>"}, nil
	}
	s.visiting[visit] = struct{}{}
	defer delete(s.visiting, visit)

	if value.Type().Key().Kind() != reflect.String {
		return map[string]any{"_value": "<unsupported-map-key>"}, nil
	}
	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	result := make(map[string]any, len(keys))
	for _, mapKey := range keys {
		name := mapKey.String()
		if s.mode == PayloadModeSafe && isSensitivePayloadKey(name) {
			s.redacted = true
			result[name] = "<redacted>"
			continue
		}
		projected, err := s.project(value.MapIndex(mapKey), name)
		if err != nil {
			return nil, err
		}
		result[name] = projected
	}
	return result, nil
}

func (s *_PayloadProjectionState) projectList(value reflect.Value) ([]any, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, nil
	}
	var visit _PayloadVisit
	if value.Kind() == reflect.Slice {
		visit = _PayloadVisit{kind: value.Kind(), ptr: value.Pointer()}
		if _, exists := s.visiting[visit]; exists {
			return []any{"<cycle>"}, nil
		}
		s.visiting[visit] = struct{}{}
		defer delete(s.visiting, visit)
	}
	result := make([]any, value.Len())
	for index := range value.Len() {
		projected, err := s.project(value.Index(index), "")
		if err != nil {
			return nil, err
		}
		result[index] = projected
	}
	return result, nil
}

func payloadFieldName(field reflect.StructField) (name string, embedded bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	name, _, _ = strings.Cut(tag, ",")
	if name != "" {
		return name, false, false
	}
	if field.Anonymous {
		return "", true, false
	}
	return field.Name, false, false
}

func marshalPayloadValue(value reflect.Value) (any, bool, error) {
	if !value.IsValid() || !value.CanInterface() {
		return nil, false, nil
	}
	if marshaler, ok := value.Interface().(json.Marshaler); ok {
		encoded, err := marshaler.MarshalJSON()
		if err != nil {
			return nil, true, err
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			return nil, true, err
		}
		return decoded, true, nil
	}
	if marshaler, ok := value.Interface().(encoding.TextMarshaler); ok {
		encoded, err := marshaler.MarshalText()
		if err != nil {
			return nil, true, err
		}
		if !utf8.Valid(encoded) {
			encoded = bytes.ToValidUTF8(encoded, []byte("�"))
		}
		return string(encoded), true, nil
	}
	return nil, false, nil
}

func isBinaryPayload(value reflect.Value) bool {
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return false
	}
	return value.Type().Elem().Kind() == reflect.Uint8
}

func isSensitivePayloadKey(key string) bool {
	normalized := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return -1
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, key)
	switch normalized {
	case "password", "passwd", "pwd", "token", "accesstoken", "refreshtoken", "secret",
		"authorization", "cookie", "setcookie", "apikey", "privatekey", "credential":
		return true
	default:
		return false
	}
}
