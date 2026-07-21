package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vpre"
)

var eventInfoBySkelName = map[string]*_EventInfo{}
var eventInfoByDefaultEmbeddedType = map[reflect.Type]*_EventInfo{}
var erDefaultEmbeddedTypes = map[reflect.Type]struct{}{}

func GetEventInfo(eventSkelName string) (EventInfo, bool) {
	eventInfo := eventInfoBySkelName[eventSkelName]
	return eventInfo, eventInfo != nil
}

func Register(eventSpec *EventSpec) {
	vpre.Check(isValidEventSpecType(eventSpec.Type), "invalid event spec type")

	eventInfo, ok := eventInfoBySkelName[eventSpec.SkelName]
	if !ok {
		eventInfo = &_EventInfo{
			name:               eventSpec.Name,
			skelName:           eventSpec.SkelName,
			hash:               eventSpec.Hash,
			payloadType:        eventSpec.PayloadType,
			emitterMethodName:  eventSpec.EmitterMethodName,
			listenerMethodName: eventSpec.ListenerMethodName,
		}
		eventInfoBySkelName[eventSpec.SkelName] = eventInfo
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
		registerDefaultEmbeddedTypes(eventInfo.DefaultListenerType(), eventInfo, false)
		registerDefaultEmbeddedTypes(eventInfo.DefaultERListenerType(), eventInfo, true)
	}
}

func registerDefaultEmbeddedTypes(defaultListenerType reflect.Type, eventInfo *_EventInfo, isERType bool) {
	embeddedType := defaultListenerType.Elem()
	eventInfoByDefaultEmbeddedType[embeddedType] = eventInfo
	if isERType {
		erDefaultEmbeddedTypes[embeddedType] = struct{}{}
	}
}

func getEventInfo(implType reflect.Type) (EventInfo, bool) {
	var eventInfo EventInfo
	isERType := false
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(implType) {
		info := eventInfoByDefaultEmbeddedType[embeddedType]
		if info == nil {
			continue
		}
		vpre.CheckNil(eventInfo, "multiple embedded default listener type found on %s.%s", implType.PkgPath(), implType.Name())
		eventInfo = info
		_, isERType = erDefaultEmbeddedTypes[embeddedType]
	}
	vpre.CheckNotNil(eventInfo, "no embedded default listener type found on %s.%s", implType.PkgPath(), implType.Name())
	return eventInfo, isERType
}

func RegisteredEventEmitterFactories() []any {
	var factories []any
	for _, eventInfo := range eventInfoBySkelName {
		factories = append(factories, eventInfo.EmitterCtor())
	}
	return factories
}
