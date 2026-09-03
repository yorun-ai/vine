package spec

import (
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
)

type _RegistryTestEvent struct{}

type _RegistryTestEmitter interface {
	EmitRegistryTestEvent(*_RegistryTestEvent)
}

type _RegistryTestListener interface {
	OnRegistryTestEvent(*_RegistryTestEvent)

	mustBeRegistryTestListener()
}

type _DefaultRegistryTestListener struct{}

func (*_DefaultRegistryTestListener) OnRegistryTestEvent(*_RegistryTestEvent) {}

func (*_DefaultRegistryTestListener) mustBeRegistryTestListener() {}

type _RegistryTestListenerER interface {
	OnRegistryTestEvent(*_RegistryTestEvent) ex.Error

	mustBeRegistryTestListenerER()
}

type _WrapperRegistryTestListenerER struct {
	_DefaultRegistryTestListener
	listenerImpl _RegistryTestListener
}

func newWrapperRegistryTestListenerER(listenerImpl _RegistryTestListener) _RegistryTestListenerER {
	return &_WrapperRegistryTestListenerER{listenerImpl: listenerImpl}
}

func (l *_WrapperRegistryTestListenerER) listener() _RegistryTestListener {
	if l.listenerImpl == nil {
		return &l._DefaultRegistryTestListener
	}
	return l.listenerImpl
}

func (l *_WrapperRegistryTestListenerER) OnRegistryTestEvent(event *_RegistryTestEvent) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	l.listener().OnRegistryTestEvent(event)
	return
}

func (*_WrapperRegistryTestListenerER) mustBeRegistryTestListenerER() {}

type _DefaultRegistryTestListenerER struct {
	_WrapperRegistryTestListenerER
}

func TestRegisterCombinesEmitterAndListenerBlocks(t *testing.T) {
	registry := NewRegistry()
	emitterSpec := &EventSpec{
		Type:               EventSpecTypeEmitter,
		Name:               "RegistryTestEvent",
		SkelName:           "event.spec.registryTestEvent",
		EmitterMethodName:  "EmitRegistryTestEvent",
		ListenerMethodName: "OnRegistryTestEvent",
		PayloadType:        reflect.TypeFor[*_RegistryTestEvent](),
		EmitterType:        reflect.TypeFor[_RegistryTestEmitter](),
		EmitterCtor:        func() _RegistryTestEmitter { return nil },
	}
	listenerSpec := &EventSpec{
		Type:                  EventSpecTypeListener,
		Name:                  "RegistryTestEvent",
		SkelName:              "event.spec.registryTestEvent",
		EmitterMethodName:     "EmitRegistryTestEvent",
		ListenerMethodName:    "OnRegistryTestEvent",
		PayloadType:           reflect.TypeFor[*_RegistryTestEvent](),
		ListenerType:          reflect.TypeFor[_RegistryTestListener](),
		DefaultListenerType:   reflect.TypeFor[*_DefaultRegistryTestListener](),
		ERListenerType:        reflect.TypeFor[_RegistryTestListenerER](),
		WrapperERListenerCtor: newWrapperRegistryTestListenerER,
		DefaultERListenerType: reflect.TypeFor[*_DefaultRegistryTestListenerER](),
	}

	registry.Register(emitterSpec)
	registry.Register(listenerSpec)

	eventInfo, ok := registry.GetEventInfo("event.spec.registryTestEvent")
	if !ok {
		t.Fatal("event info not registered")
	}
	if emitterSpec.Info() != eventInfo || listenerSpec.Info() != eventInfo {
		t.Fatal("split specs should point to the same event info")
	}
	if eventInfo.EmitterType() != reflect.TypeFor[_RegistryTestEmitter]() {
		t.Fatalf("unexpected emitter type: %v", eventInfo.EmitterType())
	}
	if eventInfo.ListenerType() != reflect.TypeFor[_RegistryTestListener]() {
		t.Fatalf("unexpected listener type: %v", eventInfo.ListenerType())
	}
	if eventInfo.ERListenerType() != reflect.TypeFor[_RegistryTestListenerER]() {
		t.Fatalf("unexpected er listener type: %v", eventInfo.ERListenerType())
	}
}

func TestRegisterRejectsDuplicateEmitterBlock(t *testing.T) {
	registry := NewRegistry()
	register := func() {
		registry.Register(&EventSpec{
			Type:               EventSpecTypeEmitter,
			Name:               "RegistryDuplicateEmitterEvent",
			SkelName:           "event.spec.registryDuplicateEmitterEvent",
			EmitterMethodName:  "EmitRegistryTestEvent",
			ListenerMethodName: "OnRegistryTestEvent",
			PayloadType:        reflect.TypeFor[*_RegistryTestEvent](),
			EmitterType:        reflect.TypeFor[_RegistryTestEmitter](),
			EmitterCtor:        func() _RegistryTestEmitter { return nil },
		})
	}
	register()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate emitter panic")
		}
		if !strings.Contains(recovered.(error).Error(), "event event.spec.registryDuplicateEmitterEvent emitter already registered") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	register()
}
