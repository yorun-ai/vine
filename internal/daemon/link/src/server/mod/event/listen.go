package event

import (
	"context"
	"errors"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
)

var newAppEventServiceClient = func(ctx context.Context, clientApp runtime.App, endpoint string, msg eventspec.NATSMessage) appskeled.EventServiceClientER {
	trace, err := meta.NewTrace(msg.Metadata.TraceId, msg.Metadata.TraceSpan)
	ex.PanicIfError(err)
	actor := meta.NewAbsentActor()
	rpcCtx := meta.NewContext(ctx, trace, nil, actor)
	return appskeled.NewEventServiceClientER(rpcclient.New(rpcclient.Option{
		Context:             rpcCtx,
		ClientApp:           clientApp,
		Logger:              logger.NewLogger(logger.GlobalOption()),
		ReturnIfSystemError: true,
		ServerEndpoint:      endpoint,
	}))
}

var runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
	return client.OnEvent(on, rpcclient.WithTimeout(timeout))
}

func (m *Manager) OnSetup(instance *minder.AppInstance) {
	if len(instance.EventListeners) == 0 {
		return
	}

	listeners := make([]*_EventListenerState, 0, len(instance.EventListeners))
	for _, registration := range instance.EventListeners {
		listenerState := &_EventListenerState{
			instance:      instance,
			appInstanceID: instance.AppInfo.InstanceId(),
			eventEndpoint: instance.EventEndpoint,
			registration:  registration,
			semaphore:     make(chan struct{}, registration.Concurrency),
		}
		listenerState.consumeContext = m.NATSClient.Consume(
			eventStreamConfig(),
			eventspec.NATSSubject(registration.EventSkelName),
			eventspec.NATSConsumerName(registration.EventSkelName, instance.AppInfo.Name()),
			m.newEventMessageHandler(listenerState),
		)
		listeners = append(listeners, listenerState)
	}

	m.mutex.Lock()
	m.listenersByAppInstanceID[instance.AppInfo.InstanceId()] = listeners
	m.mutex.Unlock()
}

func (m *Manager) OnDrain(instance *minder.AppInstance) {
	m.mutex.Lock()
	listeners := m.listenersByAppInstanceID[instance.AppInfo.InstanceId()]
	delete(m.listenersByAppInstanceID, instance.AppInfo.InstanceId())
	m.mutex.Unlock()
	for _, listenerState := range listeners {
		listenerState.consumeContext.Stop()
	}
}

func (*Manager) OnDestroy(*minder.AppInstance) {}

func (m *Manager) newEventMessageHandler(listener *_EventListenerState) func(msg jetstream.Msg) {
	return func(natsMsg jetstream.Msg) {
		msg := *vcode.MustUnmarshalJson[*eventspec.NATSMessage](natsMsg.Data())
		if !listener.instance.TryStartWork() {
			m.onEventDispatchError(natsMsg, listener.registration.NoRetry)
			return
		}
		go m.onEvent(listener, natsMsg, msg)
	}
}

func (m *Manager) onEvent(listener *_EventListenerState, natsMsg jetstream.Msg, msg eventspec.NATSMessage) {
	defer listener.instance.FinishWork()
	listener.semaphore <- struct{}{}
	defer func() {
		<-listener.semaphore
	}()

	client := newAppEventServiceClient(m.Context, m.App, listener.eventEndpoint, msg)
	on := appskeled.EventOn{
		Metadata: appskeled.EventOnMeta{
			TraceId:       msg.Metadata.TraceId,
			TraceSpan:     msg.Metadata.TraceSpan,
			AppName:       msg.Metadata.AppName,
			AppVersion:    msg.Metadata.AppVersion,
			AppInstanceId: msg.Metadata.AppInstanceId,
			EmittedAt:     msg.Metadata.EmittedAt,
		},
		EventSkelName: msg.EventSkelName,
		EventJson:     msg.EventJson,
	}

	err := runAppEvent(client, on, time.Duration(listener.registration.TimeoutMs)*time.Millisecond)
	if err != nil {
		logger.Error("event app event call failed",
			"eventSkelName", msg.EventSkelName,
			"endpoint", listener.eventEndpoint,
			"instanceId", listener.appInstanceID,
			"error", err,
		)
		m.onEventDispatchError(natsMsg, listener.registration.NoRetry)
		return
	}
	m.ackEventDispatch(natsMsg)
}

func (*Manager) ackEventDispatch(natsMsg jetstream.Msg) {
	ackNATSMessage(natsMsg.Ack())
}

func (*Manager) onEventDispatchError(natsMsg jetstream.Msg, noRetry bool) {
	var err error
	if noRetry {
		err = natsMsg.Term()
	} else {
		err = natsMsg.Nak()
	}
	ackNATSMessage(err)
}

func ackNATSMessage(err error) {
	if err == nil || errors.Is(err, nats.ErrConnectionClosed) || errors.Is(err, nats.ErrConnectionDraining) {
		return
	}
	ex.PanicIfError(err)
}
