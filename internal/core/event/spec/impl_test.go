package spec

import (
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
)

type _ImplTestEvent struct{}

type _ImplTestListener interface {
	OnImplTestEvent(*_ImplTestEvent)

	mustBeImplTestListener()
}

type _ImplTestEmitter interface {
	EmitImplTestEvent(*_ImplTestEvent)
}

type _DefaultImplTestListener struct{}

func (*_DefaultImplTestListener) OnImplTestEvent(*_ImplTestEvent) {}

func (*_DefaultImplTestListener) mustBeImplTestListener() {}

type _ImplTestListenerER interface {
	OnImplTestEvent(*_ImplTestEvent) ex.Error

	mustBeImplTestListenerER()
}

type _WrapperImplTestListenerER struct {
	_DefaultImplTestListener
	listenerImpl _ImplTestListener
}

func newWrapperImplTestListenerER(listenerImpl _ImplTestListener) _ImplTestListenerER {
	return &_WrapperImplTestListenerER{listenerImpl: listenerImpl}
}

func (l *_WrapperImplTestListenerER) listener() _ImplTestListener {
	if l.listenerImpl == nil {
		return &l._DefaultImplTestListener
	}
	return l.listenerImpl
}

func (l *_WrapperImplTestListenerER) OnImplTestEvent(event *_ImplTestEvent) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	l.listener().OnImplTestEvent(event)
	return
}

func (*_WrapperImplTestListenerER) mustBeImplTestListenerER() {}

type _DefaultImplTestListenerER struct {
	_WrapperImplTestListenerER
}

type _ImplTestEventListener struct {
	_DefaultImplTestListener
}

func (*_ImplTestEventListener) OnImplTestEvent(*_ImplTestEvent) {}

type _ImplTestEventListenerER struct {
	_DefaultImplTestListenerER
}

func (*_ImplTestEventListenerER) OnImplTestEvent(*_ImplTestEvent) ex.Error {
	return nil
}

type _InvalidImplTestEventListenerValueType struct {
	_DefaultImplTestListener
}

func (_InvalidImplTestEventListenerValueType) OnImplTestEvent(*_ImplTestEvent) {}

type _ImplTestEmitterImpl struct{}

func (*_ImplTestEmitterImpl) EmitImplTestEvent(*_ImplTestEvent) {}

var _implTestEventSpec = &EventSpec{
	Type:                  EventSpecTypeBoth,
	Name:                  "ImplTestEvent",
	SkelName:              "event.spec.implTestEvent",
	EmitterMethodName:     "EmitImplTestEvent",
	ListenerMethodName:    "OnImplTestEvent",
	PayloadType:           reflect.TypeFor[*_ImplTestEvent](),
	EmitterType:           reflect.TypeOf((*_ImplTestEmitter)(nil)).Elem(),
	EmitterCtor:           func() _ImplTestEmitter { return &_ImplTestEmitterImpl{} },
	ListenerType:          reflect.TypeOf((*_ImplTestListener)(nil)).Elem(),
	DefaultListenerType:   reflect.TypeOf(&_DefaultImplTestListener{}),
	ERListenerType:        reflect.TypeOf((*_ImplTestListenerER)(nil)).Elem(),
	WrapperERListenerCtor: newWrapperImplTestListenerER,
	DefaultERListenerType: reflect.TypeOf(&_DefaultImplTestListenerER{}),
}

func init() {
	Register(_implTestEventSpec)
}

func TestImplDictAddRegistersEvent(t *testing.T) {
	dict := NewListenerImplDict()

	dict.Add(reflect.TypeOf(&_ImplTestEventListener{}))

	listenerImpl, err := dict.GetListenerImpl(_implTestEventSpec.SkelName)
	if err != nil {
		t.Fatalf("GetListenerImpl() error = %v", err)
	}
	if listenerImpl.Info().SkelName() != _implTestEventSpec.SkelName {
		t.Fatalf("unexpected event info: %#v", listenerImpl.Info())
	}
	if listenerImpl.Type() != reflect.TypeOf(&_ImplTestEventListener{}) {
		t.Fatalf("unexpected impl type: %v", listenerImpl.Type())
	}
	if listenerImpl.IsERType() {
		t.Fatal("expected normal listener impl")
	}
}

func TestImplDictAddRegistersEREvent(t *testing.T) {
	dict := NewListenerImplDict()

	dict.Add(reflect.TypeOf(&_ImplTestEventListenerER{}))

	listenerImpl, err := dict.GetListenerImpl(_implTestEventSpec.SkelName)
	if err != nil {
		t.Fatalf("GetListenerImpl() error = %v", err)
	}
	if !listenerImpl.IsERType() {
		t.Fatal("expected er listener impl")
	}
}

func TestImplDictAddRejectsNonPointerStruct(t *testing.T) {
	dict := NewListenerImplDict()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected invalid impl type panic")
		}
		if !strings.Contains(recovered.(error).Error(), "event listener impl type spec._InvalidImplTestEventListenerValueType must be a pointer to struct") {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	dict.Add(reflect.TypeOf(_InvalidImplTestEventListenerValueType{}))
}
