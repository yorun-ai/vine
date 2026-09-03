package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

type Registry struct {
	eventInfoBySkelName            map[string]*_EventInfo
	eventInfoByDefaultEmbeddedType map[reflect.Type]*_EventInfo
	erDefaultEmbeddedTypes         map[reflect.Type]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		eventInfoBySkelName:            map[string]*_EventInfo{},
		eventInfoByDefaultEmbeddedType: map[reflect.Type]*_EventInfo{},
		erDefaultEmbeddedTypes:         map[reflect.Type]struct{}{},
	}
}

var defaultRegistry = NewRegistry()

func GetEventInfo(eventSkelName string) (EventInfo, bool) {
	return defaultRegistry.GetEventInfo(eventSkelName)
}

func (r *Registry) GetEventInfo(eventSkelName string) (EventInfo, bool) {
	eventInfo := r.eventInfoBySkelName[eventSkelName]
	return eventInfo, eventInfo != nil
}

func Register(eventSpec *EventSpec) {
	defaultRegistry.Register(eventSpec)
}

func (r *Registry) Register(eventSpec *EventSpec) {
	vpre.Check(isValidEventSpecType(eventSpec.Type), "invalid event spec type")

	eventInfo, ok := r.eventInfoBySkelName[eventSpec.SkelName]
	if !ok {
		eventInfo = &_EventInfo{
			name:               eventSpec.Name,
			skelName:           eventSpec.SkelName,
			hash:               eventSpec.Hash,
			payloadType:        eventSpec.PayloadType,
			emitterMethodName:  eventSpec.EmitterMethodName,
			listenerMethodName: eventSpec.ListenerMethodName,
		}
		r.eventInfoBySkelName[eventSpec.SkelName] = eventInfo
	}
	eventSpec.info = eventInfo

	if eventSpec.Type.setEmitter() {
		vpre.Check(!eventInfo.emitterRegistered, "event %s emitter already registered", eventSpec.SkelName)
		eventInfo.emitterRegistered = true
		eventInfo.emitterType = eventSpec.EmitterType
		eventInfo.emitterCtor = eventSpec.EmitterCtor
	}

	if eventSpec.Type.setListener() {
		vpre.Check(!eventInfo.listenerRegistered, "event %s listener already registered", eventSpec.SkelName)
		eventInfo.listenerRegistered = true
		eventInfo.listenerType = eventSpec.ListenerType
		eventInfo.defaultListenerType = eventSpec.DefaultListenerType
		eventInfo.erListenerType = eventSpec.ERListenerType
		eventInfo.wrapperERListenerCtor = eventSpec.WrapperERListenerCtor
		eventInfo.defaultERListenerType = eventSpec.DefaultERListenerType
		r.registerDefaultEmbeddedTypes(eventInfo.DefaultListenerType(), eventInfo, false)
		r.registerDefaultEmbeddedTypes(eventInfo.DefaultERListenerType(), eventInfo, true)
	}
}

func (r *Registry) registerDefaultEmbeddedTypes(defaultListenerType reflect.Type, eventInfo *_EventInfo, isERType bool) {
	embeddedType := defaultListenerType.Elem()
	r.eventInfoByDefaultEmbeddedType[embeddedType] = eventInfo
	if isERType {
		r.erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func getEventInfo(implType reflect.Type) (EventInfo, bool) {
	return defaultRegistry.getEventInfo(implType)
}

func (r *Registry) getEventInfo(implType reflect.Type) (EventInfo, bool) {
	var eventInfo EventInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		info := r.eventInfoByDefaultEmbeddedType[embeddedType]
		if info == nil {
			continue
		}
		vpre.CheckNil(eventInfo, "multiple embedded default listener type found on %s.%s", implType.PkgPath(), implType.Name())
		eventInfo = info
		_, isERType = r.erDefaultEmbeddedTypes[embeddedType]
	}
	vpre.CheckNotNil(eventInfo, "no embedded default listener type found on %s.%s", implType.PkgPath(), implType.Name())
	return eventInfo, isERType
}

func RegisteredEventEmitterFactories() []any {
	return defaultRegistry.RegisteredEventEmitterFactories()
}

func (r *Registry) RegisteredEventEmitterFactories() []any {
	var factories []any
	for _, eventInfo := range r.eventInfoBySkelName {
		factories = append(factories, eventInfo.EmitterCtor())
	}
	return factories
}
