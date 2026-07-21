package spec

import "reflect"

type EventSpecType string

const (
	EventSpecTypeListener EventSpecType = "listener"
	EventSpecTypeEmitter  EventSpecType = "emitter"
	EventSpecTypeBoth     EventSpecType = "both"
)

func isValidEventSpecType(eventSpecType EventSpecType) bool {
	return eventSpecType == EventSpecTypeListener ||
		eventSpecType == EventSpecTypeEmitter ||
		eventSpecType == EventSpecTypeBoth
}

func (et EventSpecType) setListener() bool {
	return et == EventSpecTypeListener || et == EventSpecTypeBoth
}

func (et EventSpecType) setEmitter() bool {
	return et == EventSpecTypeEmitter || et == EventSpecTypeBoth
}

type EventInfo interface {
	Name() string
	SkelName() string
	Hash() string

	PayloadType() reflect.Type

	EmitterMethodName() string
	EmitterType() reflect.Type
	EmitterCtor() any

	ListenerMethodName() string
	ListenerType() reflect.Type
	DefaultListenerType() reflect.Type
	ERListenerType() reflect.Type
	WrapperERListenerCtor() any
	DefaultERListenerType() reflect.Type

	NewEvent() any
}

type EventSpec struct {
	Type EventSpecType

	Name        string
	SkelName    string
	Hash        string
	PayloadType reflect.Type

	EmitterMethodName  string
	ListenerMethodName string

	EmitterType           reflect.Type
	EmitterCtor           any
	ListenerType          reflect.Type
	DefaultListenerType   reflect.Type
	ERListenerType        reflect.Type
	WrapperERListenerCtor any
	DefaultERListenerType reflect.Type

	info *_EventInfo
}

func (s *EventSpec) Info() EventInfo {
	return s.info
}

type _EventInfo struct {
	name        string
	skelName    string
	hash        string
	payloadType reflect.Type

	emitterRegistered bool
	emitterMethodName string
	emitterType       reflect.Type
	emitterCtor       any

	listenerRegistered    bool
	listenerMethodName    string
	listenerType          reflect.Type
	defaultListenerType   reflect.Type
	erListenerType        reflect.Type
	wrapperERListenerCtor any
	defaultERListenerType reflect.Type
}

func (ei *_EventInfo) Name() string {
	return ei.name
}

func (ei *_EventInfo) SkelName() string {
	return ei.skelName
}

func (ei *_EventInfo) Hash() string {
	return ei.hash
}

func (ei *_EventInfo) PayloadType() reflect.Type {
	return ei.payloadType
}

func (ei *_EventInfo) EmitterMethodName() string {
	return ei.emitterMethodName
}

func (ei *_EventInfo) EmitterType() reflect.Type {
	return ei.emitterType
}

func (ei *_EventInfo) EmitterCtor() any {
	return ei.emitterCtor
}

func (ei *_EventInfo) ListenerMethodName() string {
	return ei.listenerMethodName
}

func (ei *_EventInfo) ListenerType() reflect.Type {
	return ei.listenerType
}

func (ei *_EventInfo) DefaultListenerType() reflect.Type {
	return ei.defaultListenerType
}

func (ei *_EventInfo) ERListenerType() reflect.Type {
	return ei.erListenerType
}

func (ei *_EventInfo) WrapperERListenerCtor() any {
	return ei.wrapperERListenerCtor
}

func (ei *_EventInfo) DefaultERListenerType() reflect.Type {
	return ei.defaultERListenerType
}

func (ei *_EventInfo) NewEvent() any {
	return reflect.New(ei.payloadType).Interface()
}
