package event

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/event/spec"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
)

type testMessageRuntimeApp struct {
	name       string
	version    string
	instanceID string
}

func (a testMessageRuntimeApp) Name() string {
	return a.name
}

func (a testMessageRuntimeApp) Version() string {
	return a.version
}

func (a testMessageRuntimeApp) InstanceId() string {
	return a.instanceID
}

type testEmitterEvent struct {
	GroupId int `json:"groupId"`
}

type testEmitterClient struct {
	emit linkskeled.EventEmission
}

func (c *testEmitterClient) EmitEvent(emit linkskeled.EventEmission, _ivOpts ...rpcclient.InvokeOption) {
	c.emit = emit
}

var registerEmitterEventOnce = func() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			spec.Register(&spec.EventSpec{
				Type:               spec.EventSpecTypeEmitter,
				Name:               "TestEmitterEvent",
				SkelName:           "test.event.TestEmitterEvent",
				EmitterMethodName:  "EmitTestEmitter",
				ListenerMethodName: "OnTestEmitter",
				PayloadType:        reflect.TypeOf(testEmitterEvent{}),
				EmitterType:        reflect.TypeOf((*struct{ mustBe bool })(nil)).Elem(),
				EmitterCtor:        func(*Emitter) struct{ mustBe bool } { return struct{ mustBe bool }{} },
			})
		})
	}
}()

func ensureEmitterEventRegistered() {
	registerEmitterEventOnce()
}

func TestEmitterEmitUsesEventClient(t *testing.T) {
	ensureEmitterEventRegistered()
	trace := meta.InitialTrace()
	actor := meta.NewAnonymousActor()
	ctx := meta.NewContext(context.Background(), trace, nil, actor)
	app, err := meta.NewApp("demo.app", "1.0.0", "00000000-0000-0000-0000-000000000123")
	assert.NoError(t, err)
	client := &testEmitterClient{}

	emitter := NewEmitter(EmitterOption{
		Context:     ctx,
		ClientApp:   app,
		Logger:      logger.NewLogger(logger.GlobalOption()),
		EventClient: client,
	})
	eventInfo, ok := spec.GetEventInfo("test.event.TestEmitterEvent")
	assert.True(t, ok)
	assert.Equal(t, "EmitTestEmitter", eventInfo.EmitterMethodName())

	emitter.Emit(eventInfo, &testEmitterEvent{GroupId: 7})

	assert.Equal(t, "test.event.TestEmitterEvent", client.emit.EventSkelName)
	assert.Equal(t, `{"groupId":7}`, client.emit.EventJson)
	assert.Equal(t, "demo.app", client.emit.Metadata.AppName)
	assert.Equal(t, "1.0.0", client.emit.Metadata.AppVersion)
	assert.Equal(t, "00000000-0000-0000-0000-000000000123", client.emit.Metadata.AppInstanceId.String())
}
