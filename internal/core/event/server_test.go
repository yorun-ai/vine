package event

import (
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"uuid"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
)

type testServerEmitter interface {
	mustBeTestServerEmitter()
}

type defaultTestServerEmitter struct{}

func (*defaultTestServerEmitter) mustBeTestServerEmitter() {}

func testEventServerApp() meta.App {
	return meta.MustNewApp("test.app", "1.0.0", "123e4567-e89b-12d3-a456-426614174010")
}

func TestNewServerRequiresApp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing App to panic")
		}
	}()
	NewServer(Option{})
}

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

type _ProfileLabelExecutor struct {
	labels map[string]string
}

func (*_ProfileLabelExecutor) Init(spec.ListenerImplDict) {}

func (e *_ProfileLabelExecutor) Execute(ctx spec.Context, _ spec.ListenerImpl, _ any) ex.Error {
	for _, key := range []string{"vine.app", "vine.protocol", "vine.event"} {
		if value, ok := pprof.Label(ctx, key); ok {
			e.labels[key] = value
		}
	}
	return nil
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
				PayloadType:           reflect.TypeFor[testServerEvent](),
				EmitterType:           reflect.TypeFor[testServerEmitter](),
				EmitterCtor:           func(*Emitter) testServerEmitter { return &defaultTestServerEmitter{} },
				ListenerType:          reflect.TypeFor[testServerListener](),
				DefaultListenerType:   reflect.TypeFor[*defaultTestServerListener](),
				ERListenerType:        reflect.TypeFor[testServerListenerER](),
				WrapperERListenerCtor: newWrapperTestServerListenerER,
				DefaultERListenerType: reflect.TypeFor[*defaultTestServerListenerER](),
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

	logPath := filepath.Join(t.TempDir(), "event-lifecycle.jsonl")
	server := NewServer(Option{
		App:               testEventServerApp(),
		ListenerImplTypes: []reflect.Type{reflect.TypeFor[*testServerListenerImpl]()},
		Executor:          NewContainerExecutor(nil, nil),
		Logger: logger.New("vine:test", logger.WithOption{
			Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: logPath,
		}),
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

	logBytes, readErr := os.ReadFile(logPath)
	assert.NoError(t, readErr)
	lines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")
	assert.Len(t, lines, 2)
	var started map[string]any
	assert.NoError(t, json.Unmarshal([]byte(lines[0]), &started))
	assert.Equal(t, "event listener handle started", started["msg"])
	assert.Equal(t, "DEBUG", started["level"])
	assert.Contains(t, started["eventPayload"], `"groupId":9`)
	var finished map[string]any
	assert.NoError(t, json.Unmarshal([]byte(lines[1]), &finished))
	assert.Equal(t, "event listener handle finished", finished["msg"])
	assert.Equal(t, "OK", finished["code"])
	_, repeatsPayload := finished["eventPayload"]
	assert.False(t, repeatsPayload)
}

func TestServerOnEventAddsProfileLabels(t *testing.T) {
	ensureServerEventRegistered()
	executor := &_ProfileLabelExecutor{labels: map[string]string{}}
	server := NewServer(Option{
		App:               testEventServerApp(),
		ListenerImplTypes: []reflect.Type{reflect.TypeFor[*testServerListenerImpl]()},
		Executor:          executor,
	})
	trace := meta.InitialTrace()

	err := server.OnEvent(context.Background(), appskeled.EventOn{
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

	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"vine.app":      "test.app",
		"vine.protocol": "event",
		"vine.event":    "test.event.TestServerEvent",
	}, executor.labels)
}

func BenchmarkServerOnEvent(b *testing.B) {
	ensureServerEventRegistered()
	trace := meta.InitialTrace()
	server := NewServer(Option{
		App:               testEventServerApp(),
		ListenerImplTypes: []reflect.Type{reflect.TypeFor[*testServerListenerImpl]()},
		Executor:          NewContainerExecutor(nil, nil),
		Logger: logger.New("vine:benchmark", logger.WithOption{
			Level: logger.LevelError,
		}),
	})
	on := appskeled.EventOn{
		Metadata: appskeled.EventOnMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       "remote.app",
			AppVersion:    "1.0.0",
			AppInstanceId: skel.NewUUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
		},
		EventSkelName: "test.event.TestServerEvent",
		EventJson:     `{"groupId":9}`,
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := server.OnEvent(context.Background(), on); err != nil {
			b.Fatal(err)
		}
	}
}

func TestServerRejectedUsesMainEventFieldNames(t *testing.T) {
	ensureServerEventRegistered()
	logPath := filepath.Join(t.TempDir(), "event-rejected.jsonl")
	server := NewServer(Option{
		App:               testEventServerApp(),
		ListenerImplTypes: []reflect.Type{reflect.TypeFor[*testServerListenerImpl]()},
		Executor:          NewContainerExecutor(nil, nil),
		Logger: logger.New("vine:test", logger.WithOption{
			Format: logger.FormatJSON, Level: logger.LevelDebug, OutputPath: logPath,
		}),
	})

	err := server.OnEvent(context.Background(), appskeled.EventOn{EventSkelName: "missing.event"})
	if err == nil || err.Code() != ex.InvalidEvent {
		t.Fatalf("unexpected error: %#v", err)
	}

	logBytes, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read rejection log: %v", readErr)
	}
	var record map[string]any
	if decodeErr := json.Unmarshal(logBytes, &record); decodeErr != nil {
		t.Fatalf("decode rejection log: %v", decodeErr)
	}
	if record["msg"] != "event listener handle rejected" || record["level"] != "ERROR" {
		t.Fatalf("unexpected rejection record: %#v", record)
	}
	if record["eventSkel"] != "missing.event" {
		t.Fatalf("rejection record must preserve main Event field names: %#v", record)
	}
	if _, exists := record["eventSkelName"]; exists {
		t.Fatalf("rejection record must not emit renamed Event fields: %#v", record)
	}
}
