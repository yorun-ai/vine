package app

import (
	"context"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/event"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	"go.yorun.ai/vine/internal/core/skel"
)

type testEventerSpec struct {
	Application
	EventerEnabled
}

func (*testEventerSpec) Name() string {
	return "test.eventer"
}

func (*testEventerSpec) EventerBind(*di.Binder) {}

func (*testEventerSpec) EventerInitListeners(addListener ListenerTypeAdder) {
	addListener(
		T[*testEventerListenerImpl](),
		WithListenerTimeout(time.Second),
		WithListenerConcurrency(2),
		WithListenerNoRetry(),
	)
}

func (*testEventerSpec) EventerInitFilters(addFilter TypeAdder) {
	addFilter(T[*testEventerFilter]())
}

type testEventerFilter struct{}

func (*testEventerFilter) Filter(next ctr.FilterNext) {
	next()
}

type testEventerEmitter interface {
	mustBeTestEventerEmitter()
}

type defaultTestEventerEmitter struct{}

func (*defaultTestEventerEmitter) mustBeTestEventerEmitter() {}

type testEventerListener interface {
	OnTestEventer(event *testEventerEvent)
	mustBeTestEventerListener()
}

type defaultTestEventerListener struct{}

func (*defaultTestEventerListener) OnTestEventer(event *testEventerEvent) {}
func (*defaultTestEventerListener) mustBeTestEventerListener()            {}

type testEventerListenerER interface {
	OnTestEventer(event *testEventerEvent) ex.Error
	mustBeTestEventerListenerER()
}

type _WrapperTestEventerListenerER struct {
	defaultTestEventerListener
	listenerImpl testEventerListener
}

func newWrapperTestEventerListenerER(listenerImpl testEventerListener) testEventerListenerER {
	return &_WrapperTestEventerListenerER{listenerImpl: listenerImpl}
}

func (l *_WrapperTestEventerListenerER) listener() testEventerListener {
	if l.listenerImpl == nil {
		return &l.defaultTestEventerListener
	}
	return l.listenerImpl
}

func (l *_WrapperTestEventerListenerER) OnTestEventer(event *testEventerEvent) (err ex.Error) {
	defer func() { err = ex.Recover(recover()) }()
	l.listener().OnTestEventer(event)
	return
}

func (*_WrapperTestEventerListenerER) mustBeTestEventerListenerER() {}

type defaultTestEventerListenerER struct {
	_WrapperTestEventerListenerER
}

type testEventerListenerImpl struct {
	defaultTestEventerListener
}

type testEventerEvent struct {
	GroupId int `json:"groupId"`
}

var testEventerOnGroupID int

func (*testEventerListenerImpl) OnTestEventer(event *testEventerEvent) {
	testEventerOnGroupID = event.GroupId
}

var registerEventerEventOnce = func() func() {
	var done bool
	return func() {
		if done {
			return
		}
		done = true
		eventspec.Register(&eventspec.EventSpec{
			Type:                  eventspec.EventSpecTypeBoth,
			Name:                  "TestEventerEvent",
			SkelName:              "test.eventer.TestEventerEvent",
			ListenerMethodName:    "OnTestEventer",
			PayloadType:           reflect.TypeOf(testEventerEvent{}),
			EmitterType:           reflect.TypeOf((*testEventerEmitter)(nil)).Elem(),
			EmitterCtor:           func(*event.Emitter) testEventerEmitter { return &defaultTestEventerEmitter{} },
			ListenerType:          reflect.TypeOf((*testEventerListener)(nil)).Elem(),
			DefaultListenerType:   reflect.TypeOf(&defaultTestEventerListener{}),
			ERListenerType:        reflect.TypeOf((*testEventerListenerER)(nil)).Elem(),
			WrapperERListenerCtor: newWrapperTestEventerListenerER,
			DefaultERListenerType: reflect.TypeOf(&defaultTestEventerListenerER{}),
		})
	}
}()

func ensureEventerEventRegistered() {
	registerEventerEventOnce()
}

func TestNewEventerStoresAppAndSpec(t *testing.T) {
	ensureEventerEventRegistered()
	app := newTestAppImpl()
	spec := &testEventerSpec{}

	m := newEventer(spec, app.info, app.bindAppDeps)

	assert.Equal(t, app.info, m.appInfo)
	assert.Same(t, spec, m.spec)
}

func TestNewEventerInitBuildsServers(t *testing.T) {
	ensureEventerEventRegistered()

	app := newTestAppImpl()
	spec := &testEventerSpec{}

	eventer := newEventer(spec, app.info, app.bindAppDeps)

	assert.NotNil(t, eventer.eventServer)
	assert.NotNil(t, eventer.rpcServer)
	assert.Equal(t, []reflect.Type{T[*testEventerListenerImpl]()}, eventer.listenerTypes())
	assert.Equal(t, []reflect.Type{T[*testEventerFilter]()}, eventer.filterTypes())
	assert.Len(t, eventer.listeners, 1)
	assert.Equal(t, time.Second, eventer.listeners[0].options.Timeout)
	assert.Equal(t, 2, eventer.listeners[0].options.Concurrency)
	assert.True(t, eventer.listeners[0].options.NoRetry)
}

func TestNewEventerUsesDefaultListenerOptions(t *testing.T) {
	ensureEventerEventRegistered()

	app := newTestAppImpl()
	spec := &testFullSpec{}

	eventer := newEventer(spec, app.info, app.bindAppDeps)

	assert.Len(t, eventer.listeners, 1)
	assert.Equal(t, 30*time.Second, eventer.listeners[0].options.Timeout)
	assert.Equal(t, 10, eventer.listeners[0].options.Concurrency)
	assert.False(t, eventer.listeners[0].options.NoRetry)
}

func TestAppEventServiceServerOnEventForwardsToEventServer(t *testing.T) {
	ensureEventerEventRegistered()
	testEventerOnGroupID = 0

	app := newTestAppImpl()
	eventer := newEventer(&testEventerSpec{}, app.info, app.bindAppDeps)
	actor := meta.NewAbsentActor()
	service := &_AppEventServiceServerImpl{
		Context:     rpcspec.NewContext(context.Background(), meta.InitialTrace(), nil, nil, actor),
		EventServer: eventer.eventServer,
	}

	service.OnEvent(appskeled.EventOn{
		Metadata: appskeled.EventOnMeta{
			TraceId:       meta.InitialTrace().Id(),
			TraceSpan:     meta.InitialTrace().Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		EventSkelName: "test.eventer.TestEventerEvent",
		EventJson:     `{"groupId":7}`,
	})

	assert.Equal(t, 7, testEventerOnGroupID)
}
