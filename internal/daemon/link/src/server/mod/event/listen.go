package event

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	linknats "go.yorun.ai/vine/internal/daemon/link/src/server/comp/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
	"go.yorun.ai/vine/util/vcode"
)

var eventLogger = logger.New("daemon:link:event")

var newAppEventServiceClient = func(ctx context.Context, clientApp meta.App, endpoint string, msg eventspec.NATSMessage) appskeled.EventServiceClientER {
	trace, err := meta.NewTrace(msg.Metadata.TraceId, msg.Metadata.TraceSpan)
	ex.PanicIfError(err)
	actor := meta.NewAbsentActor()
	rpcCtx := meta.NewContext(ctx, trace, nil, actor)
	return appskeled.NewEventServiceClientER(rpcclient.New(rpcclient.Option{
		Context:             rpcCtx,
		ClientApp:           clientApp,
		Logger:              eventLogger.Child("client"),
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.listenersByAppInstanceID == nil {
		return
	}
	id := instance.AppInfo.InstanceId()
	for _, registration := range instance.EventListeners {
		messages := m.NATSClient.Messages(
			m.Context,
			eventStreamConfig(),
			eventspec.NATSSubject(registration.EventSkelName),
			eventspec.NATSConsumerName(registration.EventSkelName, instance.AppInfo.Name()),
		)
		ctx, cancel := context.WithCancel(m.Context)
		listener := &_EventListenerState{
			instance:      instance,
			appInstanceID: id,
			eventEndpoint: instance.EventEndpoint,
			registration:  registration,
			semaphore:     make(chan struct{}, registration.Concurrency),
			cancelRead:    cancel,
			done:          make(chan struct{}),
		}
		m.listenersByAppInstanceID[id] = append(m.listenersByAppInstanceID[id], listener)
		go m.dispatchEvents(ctx, listener, messages)
	}
}

func (m *Manager) OnDrain(instance *minder.AppInstance) {
	m.mutex.Lock()
	listeners := m.listenersByAppInstanceID[instance.AppInfo.InstanceId()]
	delete(m.listenersByAppInstanceID, instance.AppInfo.InstanceId())
	m.mutex.Unlock()
	stopEventListeners(listeners)
}

func (*Manager) OnDestroy(*minder.AppInstance) {}

func (m *Manager) dispatchEvents(ctx context.Context, listener *_EventListenerState, messages linknats.MessagesContext) {
	defer close(listener.done)
	dispatch := linknats.MessageDispatcher[_EventListenerState]{
		Reserve: func() (*_EventListenerState, context.Context) {
			select {
			case <-ctx.Done():
				return nil, nil
			case listener.semaphore <- struct{}{}:
				return listener, ctx
			}
		},
		Admit: func(target *_EventListenerState, msg jetstream.Msg) bool {
			if ctx.Err() == nil && target.instance.TryStartWork() {
				return true
			}
			target.cancelRead()
			m.onEventDispatchError(msg, target.registration.NoRetry)
			return false
		},
		Handle:  m.onEvent,
		Release: func(target *_EventListenerState) { <-target.semaphore },
	}
	dispatch.Run(ctx, messages)
}

func (m *Manager) onEvent(listener *_EventListenerState, natsMsg jetstream.Msg) {
	defer listener.instance.FinishWork()
	msg := *vcode.MustUnmarshalJson[*eventspec.NATSMessage](natsMsg.Data())
	err := m.runEvent(listener, msg)
	if err != nil {
		m.onEventDispatchError(natsMsg, listener.registration.NoRetry)
		return
	}
	m.ackEventDispatch(natsMsg)
}

func (m *Manager) runEvent(listener *_EventListenerState, msg eventspec.NATSMessage) ex.Error {
	client := newAppEventServiceClient(m.Context, m.CurrentApp, listener.eventEndpoint, msg)
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
		eventLogger.Error("event app event call failed",
			"eventSkelName", msg.EventSkelName,
			"endpoint", listener.eventEndpoint,
			"instanceId", listener.appInstanceID,
			"error", err,
		)
	}
	return err
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
