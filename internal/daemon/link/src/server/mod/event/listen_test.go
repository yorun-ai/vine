package event

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appskeled "go.yorun.ai/vine/internal/core/app/skeled"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/meta"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

const testPathEvent = "/event"

func TestManagerRegistersListenerAndDispatchesEvent(t *testing.T) {
	manager, cleanup := newTestManager(t)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	hooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppEventServiceClient = func(context.Context, meta.App, string, eventspec.NATSMessage) appskeled.EventServiceClientER {
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				hooks.mutex.Lock()
				hooks.events = append(hooks.events, on)
				hooks.mutex.Unlock()
				hooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		hooks.mutex.Lock()
		hooks.timeout = timeout
		hooks.callCount++
		hooks.mutex.Unlock()
		return client.OnEvent(on)
	}

	appInfo, err := meta.NewApp("demo.app", "1.0.0", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.NoError(t, err)
	endpoint := testLocalAppEndpoint(8080)
	manager.AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       appInfo,
		EventEndpoint: endpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     2500,
			Concurrency:   1,
		}},
	})

	manager.EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	requireDispatchSignal(t, hooks.completed, "event dispatch timeout")

	hooks.mutex.Lock()
	defer hooks.mutex.Unlock()
	assert.Equal(t, time.Duration(2500)*time.Millisecond, hooks.timeout)
	assert.Equal(t, 1, hooks.callCount)
	assert.Equal(t, "demo.user.UserCreatedEvent", hooks.events[0].EventSkelName)
	assert.Equal(t, `{"userId":"u1"}`, hooks.events[0].EventJson)
	assert.Equal(t, "launcher.app", hooks.events[0].Metadata.AppName)
}

func TestBusyEventListenerLeavesBacklogForAnotherInstance(t *testing.T) {
	for _, test := range []struct{ total, capacity int }{{64, 1}, {500, 1}, {64, 3}} {
		total, capacity := test.total, test.capacity
		t.Run(fmt.Sprintf("events=%d/capacity=%d", total, capacity), func(t *testing.T) {
			const eventName = "OrderPaid"
			const appName = "orders"
			managers, cleanup := newTestManagers(t, 2)
			oldFactory := newAppEventServiceClient
			defer func() { newAppEventServiceClient = oldFactory }()
			defer cleanup()

			gate := make(chan struct{})
			release := sync.OnceFunc(func() { close(gate) })
			defer release()
			startedA := make(chan string, total)
			startedB := make(chan string, total)
			newAppEventServiceClient = func(_ context.Context, _ meta.App, endpoint string, _ eventspec.NATSMessage) appskeled.EventServiceClientER {
				return &_ManagerAppEventClient{onEvent: func(on appskeled.EventOn) error {
					if endpoint == "A" {
						startedA <- on.EventJson
						<-gate
					} else {
						startedB <- on.EventJson
					}
					return nil
				}}
			}
			listener := func(endpoint string) *minder.AppInstance {
				return &minder.AppInstance{
					AppInfo:       meta.MustNewAppWithRandomId(appName, "1.0.0"),
					EventEndpoint: endpoint,
					EventListeners: []skeled.EventListenerRegistration{{
						EventSkelName: eventName,
						TimeoutMs:     3000,
						Concurrency:   capacity,
					}},
				}
			}
			a, b := listener("A"), listener("B")
			managers[0].OnSetup(a)
			emit := func(id int) {
				managers[0].EmitEvent(skeled.EventEmission{EventSkelName: eventName, EventJson: strconv.Itoa(id)})
			}
			conn := hubnats.ConnectInproc()
			defer conn.Close()
			js, err := jetstream.New(conn)
			require.NoError(t, err)
			consumer, err := js.Consumer(t.Context(), eventspec.NATSStreamName, eventspec.NATSConsumerName(eventName, appName))
			require.NoError(t, err)
			next := func(started <-chan string) string {
				t.Helper()
				select {
				case id := <-started:
					return id
				case <-time.After(3 * time.Second):
					info, err := consumer.Info(t.Context())
					require.NoError(t, err)
					t.Fatalf("event delivery timed out: delivered=%d pending=%d ackPending=%d waitingPulls=%d",
						info.Delivered.Consumer, info.NumPending, info.NumAckPending, info.NumWaiting)
					return ""
				}
			}
			for id := range capacity {
				emit(id)
				require.Equal(t, strconv.Itoa(id), next(startedA))
			}
			var remaining []string
			for id := capacity; id < total; id++ {
				emit(id)
				remaining = append(remaining, strconv.Itoa(id))
			}

			// B can process the entire backlog while A's calls remain blocked.
			managers[1].OnSetup(b)
			var received []string
			for range remaining {
				received = append(received, next(startedB))
			}
			require.ElementsMatch(t, remaining, received)

			a.BeginDrain()
			managers[0].OnDrain(a)
			release()
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			require.True(t, a.WaitDrain(ctx))
			require.Empty(t, startedA)
			require.Eventually(t, func() bool {
				info, err := consumer.Info(ctx)
				return err == nil && info.NumAckPending == 0 && info.NumPending == 0
			}, 3*time.Second, time.Millisecond)
		})
	}
}

type longEventMessage struct {
	jetstream.Msg
	handlerDone chan struct{}
	replies     []string
}

func (*longEventMessage) Data() []byte { return []byte("{}") }

func (m *longEventMessage) InProgress() error {
	// The second send is still in flight when the handler returns at 45s.
	if len(m.replies) == 1 {
		<-m.handlerDone
	}
	m.replies = append(m.replies, "InProgress")
	return nil
}

func (m *longEventMessage) Ack() error {
	m.replies = append(m.replies, "Ack")
	return nil
}

func TestLongEventKeepsRenewingDuringDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		instance := new(minder.AppInstance)
		require.True(t, instance.TryStartWork())
		msg := new(longEventMessage{handlerDone: make(chan struct{})})
		oldFactory := newAppEventServiceClient
		t.Cleanup(func() { newAppEventServiceClient = oldFactory })
		newAppEventServiceClient = func(context.Context, meta.App, string, eventspec.NATSMessage) appskeled.EventServiceClientER {
			return &_ManagerAppEventClient{onEvent: func(appskeled.EventOn) error {
				instance.BeginDrain()
				time.Sleep(45 * time.Second)
				close(msg.handlerDone)
				return nil
			}}
		}
		manager := &Manager{Context: t.Context()}
		listener := &_EventListenerState{
			instance:     instance,
			registration: skeled.EventListenerRegistration{TimeoutMs: 60_000},
		}
		manager.onEvent(listener, msg)
		time.Sleep(30 * time.Second)
		require.Equal(t, []string{"InProgress", "InProgress", "Ack"}, msg.replies)
	})
}

// The pull was already outstanding when the instance started draining.
type drainingEventPull struct {
	instance *minder.AppInstance
	message  *unadmittedEvent
}

func (p *drainingEventPull) Next(context.Context) (jetstream.Msg, error) {
	if p.message == nil {
		return nil, jetstream.ErrMsgIteratorClosed
	}
	p.instance.BeginDrain()
	msg := p.message
	p.message = nil
	return msg, nil
}

func (*drainingEventPull) Stop() {}

type unadmittedEvent struct {
	jetstream.Msg
	reply string
}

func (m *unadmittedEvent) Nak() error {
	m.reply = "Nak"
	return nil
}

func (m *unadmittedEvent) Term() error {
	m.reply = "Term"
	return nil
}

func TestEventPullFinishingDuringDrainUsesListenerRetryPolicy(t *testing.T) {
	for _, test := range []struct {
		noRetry bool
		reply   string
	}{{false, "Nak"}, {true, "Term"}} {
		t.Run(test.reply, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			instance := new(minder.AppInstance)
			listener := &_EventListenerState{
				instance:     instance,
				cancelRead:   cancel,
				registration: skeled.EventListenerRegistration{NoRetry: test.noRetry},
				semaphore:    make(chan struct{}, 1),
				done:         make(chan struct{}),
			}
			msg := new(unadmittedEvent)
			manager := new(Manager)
			manager.dispatchEvents(ctx, listener, &drainingEventPull{instance: instance, message: msg})
			require.Equal(t, test.reply, msg.reply)
			require.Empty(t, listener.semaphore)
		})
	}
}

func TestManagerFansOutByAppName(t *testing.T) {
	managers, cleanup := newTestManagers(t, 2)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	secondHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppEventServiceClient = func(_ context.Context, _ meta.App, endpoint string, _ eventspec.NATSMessage) appskeled.EventServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				targetHooks.mutex.Lock()
				targetHooks.events = append(targetHooks.events, on)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				targetHooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		return client.OnEvent(on)
	}

	firstAppInfo, err := meta.NewApp("demo.first", "1.0.0", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.second", "1.0.0", uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	require.NoError(t, err)

	firstEndpoint := testLocalAppEndpoint(8080)
	managers[0].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       firstAppInfo,
		EventEndpoint: firstEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})
	secondEndpoint := testLocalAppEndpoint(8081)
	managers[1].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       secondAppInfo,
		EventEndpoint: secondEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})

	managers[0].EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	requireDispatchSignal(t, firstHooks.completed, "first app event dispatch timeout")
	requireDispatchSignal(t, secondHooks.completed, "second app event dispatch timeout")
}

func TestManagerCompetesAcrossSameAppNameInstances(t *testing.T) {
	managers, cleanup := newTestManagers(t, 2)
	defer cleanup()

	oldFactory := newAppEventServiceClient
	oldRun := runAppEvent
	defer func() {
		newAppEventServiceClient = oldFactory
		runAppEvent = oldRun
	}()

	firstHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	secondHooks := &_ManagerDispatchHooks{completed: make(chan struct{}, 1)}
	newAppEventServiceClient = func(_ context.Context, _ meta.App, endpoint string, _ eventspec.NATSMessage) appskeled.EventServiceClientER {
		targetHooks := secondHooks
		if strings.Contains(endpoint, ":8080") {
			targetHooks = firstHooks
		}
		return &_ManagerAppEventClient{
			onEvent: func(on appskeled.EventOn) error {
				targetHooks.mutex.Lock()
				targetHooks.events = append(targetHooks.events, on)
				targetHooks.callCount++
				targetHooks.mutex.Unlock()
				targetHooks.completed <- struct{}{}
				return nil
			},
		}
	}
	runAppEvent = func(client appskeled.EventServiceClientER, on appskeled.EventOn, timeout time.Duration) ex.Error {
		return client.OnEvent(on)
	}

	firstAppInfo, err := meta.NewApp("demo.app", "1.0.0", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.NoError(t, err)
	secondAppInfo, err := meta.NewApp("demo.app", "1.0.0", uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	require.NoError(t, err)

	firstEndpoint := testLocalAppEndpoint(8080)
	managers[0].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       firstAppInfo,
		EventEndpoint: firstEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})
	secondEndpoint := testLocalAppEndpoint(8081)
	managers[1].AppMinder.RegisterInstance(minder.AppRegistration{
		AppInfo:       secondAppInfo,
		EventEndpoint: secondEndpoint + testPathEvent,
		EventListeners: []skeled.EventListenerRegistration{{
			EventSkelName: "demo.user.UserCreatedEvent",
			TimeoutMs:     1000,
			Concurrency:   1,
		}},
	})

	managers[0].EmitEvent(skeled.EventEmission{
		Metadata: skeled.EventEmissionMeta{
			TraceId:       "trace-1",
			TraceSpan:     "0123456789abcdef",
			AppName:       "launcher.app",
			AppVersion:    "2.0.0",
			AppInstanceId: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		},
		EventSkelName: "demo.user.UserCreatedEvent",
		EventJson:     `{"userId":"u1"}`,
	})

	select {
	case <-firstHooks.completed:
	case <-secondHooks.completed:
	case <-time.After(2 * time.Second):
		t.Fatal("competing event dispatch timeout")
	}

	firstHooks.mutex.Lock()
	firstCount := firstHooks.callCount
	firstHooks.mutex.Unlock()
	secondHooks.mutex.Lock()
	secondCount := secondHooks.callCount
	secondHooks.mutex.Unlock()
	assert.Equal(t, 1, firstCount+secondCount)
}

func requireDispatchSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func testLocalAppEndpoint(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
