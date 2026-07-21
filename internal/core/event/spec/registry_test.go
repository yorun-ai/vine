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
	emitterSpec := &EventSpec{
		Type:               EventSpecTypeEmitter,
		Name:               "RegistryTestEvent",
		SkelName:           "event.spec.registryTestEvent",
		EmitterMethodName:  "EmitRegistryTestEvent",
		ListenerMethodName: "OnRegistryTestEvent",
		PayloadType:        reflect.TypeFor[*_RegistryTestEvent](),
		EmitterType:        reflect.TypeOf((*_RegistryTestEmitter)(nil)).Elem(),
		EmitterCtor:        func() _RegistryTestEmitter { return nil },
	}
	listenerSpec := &EventSpec{
		Type:                  EventSpecTypeListener,
		Name:                  "RegistryTestEvent",
		SkelName:              "event.spec.registryTestEvent",
		EmitterMethodName:     "EmitRegistryTestEvent",
		ListenerMethodName:    "OnRegistryTestEvent",
		PayloadType:           reflect.TypeFor[*_RegistryTestEvent](),
		ListenerType:          reflect.TypeOf((*_RegistryTestListener)(nil)).Elem(),
		DefaultListenerType:   reflect.TypeOf(&_DefaultRegistryTestListener{}),
		ERListenerType:        reflect.TypeOf((*_RegistryTestListenerER)(nil)).Elem(),
		WrapperERListenerCtor: newWrapperRegistryTestListenerER,
		DefaultERListenerType: reflect.TypeOf(&_DefaultRegistryTestListenerER{}),
	}

	Register(emitterSpec)
	Register(listenerSpec)

	eventInfo, ok := GetEventInfo("event.spec.registryTestEvent")
	if !ok {
		t.Fatal("event info not registered")
	}
	if emitterSpec.Info() != eventInfo || listenerSpec.Info() != eventInfo {
		t.Fatal("split specs should point to the same event info")
	}
	if eventInfo.EmitterType() != reflect.TypeOf((*_RegistryTestEmitter)(nil)).Elem() {
		t.Fatalf("unexpected emitter type: %v", eventInfo.EmitterType())
	}
	if eventInfo.ListenerType() != reflect.TypeOf((*_RegistryTestListener)(nil)).Elem() {
		t.Fatalf("unexpected listener type: %v", eventInfo.ListenerType())
	}
	if eventInfo.ERListenerType() != reflect.TypeOf((*_RegistryTestListenerER)(nil)).Elem() {
		t.Fatalf("unexpected er listener type: %v", eventInfo.ERListenerType())
	}
}

func TestRegisterRejectsDuplicateEmitterBlock(t *testing.T) {
	register := func() {
		Register(&EventSpec{
			Type:               EventSpecTypeEmitter,
			Name:               "RegistryDuplicateEmitterEvent",
			SkelName:           "event.spec.registryDuplicateEmitterEvent",
			EmitterMethodName:  "EmitRegistryTestEvent",
			ListenerMethodName: "OnRegistryTestEvent",
			PayloadType:        reflect.TypeFor[*_RegistryTestEvent](),
			EmitterType:        reflect.TypeOf((*_RegistryTestEmitter)(nil)).Elem(),
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
