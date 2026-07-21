package event

import (
	"context"
	"github.com/google/uuid"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
)

type testServerEmitter interface {
	mustBeTestServerEmitter()
}

type defaultTestServerEmitter struct{}

func (*defaultTestServerEmitter) mustBeTestServerEmitter() {}

type testServerListener interface {
	OnTestServer(event *testServerEvent)
	mustBeTestServerListener()
}

type defaultTestServerListener struct{}

func (*defaultTestServerListener) OnTestServer(event *testServerEvent) {}
func (*defaultTestServerListener) mustBeTestServerListener()           {}

type testServerListenerER interface {
	OnTestServer(event *testServerEvent) ex.Error
	mustBeTestServerListenerER()
}

type _WrapperTestServerListenerER struct {
	defaultTestServerListener
	listenerImpl testServerListener
}

func newWrapperTestServerListenerER(listenerImpl testServerListener) testServerListenerER {
	return &_WrapperTestServerListenerER{listenerImpl: listenerImpl}
}

func (l *_WrapperTestServerListenerER) listener() testServerListener {
	if l.listenerImpl == nil {
		return &l.defaultTestServerListener
	}
	return l.listenerImpl
}

func (l *_WrapperTestServerListenerER) OnTestServer(event *testServerEvent) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	l.listener().OnTestServer(event)
	return
}

func (*_WrapperTestServerListenerER) mustBeTestServerListenerER() {}

type defaultTestServerListenerER struct {
	_WrapperTestServerListenerER
}

type testServerListenerImpl struct {
	defaultTestServerListener
}

type testServerEvent struct {
	GroupId int `json:"groupId"`
}

var testServerGroupID int

func (*testServerListenerImpl) OnTestServer(event *testServerEvent) {
	testServerGroupID = event.GroupId
}

var registerServerEventOnce = func() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			spec.Register(&spec.EventSpec{
				Type:                  spec.EventSpecTypeBoth,
				Name:                  "TestServerEvent",
				SkelName:              "test.event.TestServerEvent",
				EmitterMethodName:     "EmitTestServer",
				ListenerMethodName:    "OnTestServer",
				PayloadType:           reflect.TypeOf(testServerEvent{}),
				EmitterType:           reflect.TypeOf((*testServerEmitter)(nil)).Elem(),
				EmitterCtor:           func(*Emitter) testServerEmitter { return &defaultTestServerEmitter{} },
				ListenerType:          reflect.TypeOf((*testServerListener)(nil)).Elem(),
				DefaultListenerType:   reflect.TypeOf(&defaultTestServerListener{}),
				ERListenerType:        reflect.TypeOf((*testServerListenerER)(nil)).Elem(),
				WrapperERListenerCtor: newWrapperTestServerListenerER,
				DefaultERListenerType: reflect.TypeOf(&defaultTestServerListenerER{}),
			})
		})
	}
}()

func ensureServerEventRegistered() {
	registerServerEventOnce()
}

func TestServerOnEventForwardsToListener(t *testing.T) {
	ensureServerEventRegistered()
	testServerGroupID = 0
	trace := meta.InitialTrace()

	server := NewServer(Option{
		ListenerImplTypes: []reflect.Type{reflect.TypeOf(&testServerListenerImpl{})},
		Executor:          NewContainerExecutor(nil, nil),
	})

	errI := server.OnEvent(context.Background(), appskeled.EventOn{
		Metadata: appskeled.EventOnMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		EventSkelName: "test.event.TestServerEvent",
		EventJson:     `{"groupId":9}`,
	})
	assert.Nil(t, errI)
	assert.Equal(t, 9, testServerGroupID)

	eventInfo, ok := spec.GetEventInfo("test.event.TestServerEvent")
	assert.True(t, ok)
	assert.Equal(t, "EmitTestServer", eventInfo.EmitterMethodName())
}
