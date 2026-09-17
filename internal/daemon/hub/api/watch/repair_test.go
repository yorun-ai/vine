package watch_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
)

type _EndpointTestClient struct {
	watch.Client

	endpoint string
}

func (c *_EndpointTestClient) InitOption(option *watch.Option) {
	option.Endpoint = c.endpoint
	option.Username = watch.LinkUsername
	option.Password = watch.LinkPassword
}

type _EventLog struct {
	mutex  sync.Mutex
	events []watch.Event
}

func (l *_EventLog) append(event watch.Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.events = append(l.events, event)
}

func (l *_EventLog) has(kind string, key string, value string) func() bool {
	return func() bool {
		l.mutex.Lock()
		defer l.mutex.Unlock()

		for _, event := range l.events {
			if event.Kind == kind && event.Key == key && event.Value == value {
				return true
			}
		}
		return false
	}
}

func newTestWatchServer(t *testing.T) *watchserver.Server {
	t.Helper()

	server := watchserver.NewServerForTest()
	server.Option.WatchListen = "127.0.0.1:0"
	server.DIInit()
	t.Cleanup(server.AfterAppStop)
	return server
}

func testWatchEndpoint(t *testing.T, server *watchserver.Server) string {
	t.Helper()

	addr := watchserver.WatchListenAddrForTest(t, server)
	require.NotEmpty(t, addr)
	return "redis://" + addr
}

func TestRepairEndpointResubscribesWatchersToNewHub(t *testing.T) {
	previousHub := newTestWatchServer(t)
	nextHub := newTestWatchServer(t)

	client := &_EndpointTestClient{endpoint: testWatchEndpoint(t, previousHub)}
	manager := &watch.ClientManager{Context: context.Background()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)

	events := new(_EventLog)
	valuesByKey, subscription := client.LoadListAndSubscribe(t.Context(), "rpc:test:endpoint", events.append)
	subscription.Start()
	assert.Empty(t, valuesByKey)

	previousHub.SetAndNotify("rpc:test:endpoint:demo", "one")
	require.Eventually(t, events.has(watch.EventKindUpsert, "rpc:test:endpoint:demo", "one"), 5*time.Second, 10*time.Millisecond)

	// Keep the same endpoint first: nothing may reconnect.
	unchanged := &watch.Option{}
	client.InitOption(unchanged)
	assert.False(t, client.RepairEndpoint(unchanged))

	client.endpoint = testWatchEndpoint(t, nextHub)
	option := &watch.Option{}
	client.InitOption(option)
	assert.True(t, client.RepairEndpoint(option))

	// The watcher reloaded its snapshot from the new Hub, so the key that only
	// existed on the previous Hub is reported as deleted.
	require.Eventually(t, events.has(watch.EventKindDelete, "rpc:test:endpoint:demo", ""), 5*time.Second, 10*time.Millisecond)

	nextHub.SetAndNotify("rpc:test:endpoint:other", "two")
	require.Eventually(t, events.has(watch.EventKindUpsert, "rpc:test:endpoint:other", "two"), 5*time.Second, 10*time.Millisecond)
}

func TestRepairFailureKeepsSubscriptionsAndCanRetry(t *testing.T) {
	old := newTestWatchServer(t)
	next := newTestWatchServer(t)
	client := &_EndpointTestClient{endpoint: testWatchEndpoint(t, old)}
	manager := &watch.ClientManager{Context: t.Context()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)
	events := new(_EventLog)
	_, sub := client.LoadListAndSubscribe(t.Context(), "rpc:test:endpoint", events.append)
	sub.Start()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	option := &watch.Option{Endpoint: addr, Username: watch.LinkUsername, Password: watch.LinkPassword}
	require.Panics(t, func() { client.RepairEndpoint(option) })
	old.SetAndNotify("rpc:test:endpoint:demo", "old-still-live")
	require.Eventually(t, events.has(watch.EventKindUpsert, "rpc:test:endpoint:demo", "old-still-live"), time.Second, time.Millisecond)
	option.Endpoint = testWatchEndpoint(t, next)
	require.True(t, client.RepairEndpoint(option))
	next.SetAndNotify("rpc:test:endpoint:demo", "new-live")
	require.Eventually(t, events.has(watch.EventKindUpsert, "rpc:test:endpoint:demo", "new-live"), time.Second, time.Millisecond)
}

func TestSubscribeDuringEndpointReplacement(t *testing.T) {
	old := newTestWatchServer(t)
	next := newTestWatchServer(t)
	client := &_EndpointTestClient{endpoint: testWatchEndpoint(t, old)}
	manager := &watch.ClientManager{Context: t.Context()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)
	logs := make([]*_EventLog, 20)
	var group sync.WaitGroup
	for i := range logs {
		logs[i] = new(_EventLog)
		group.Go(func() {
			_, subscription := client.LoadListAndSubscribe(t.Context(), "rpc:test:endpoint", logs[i].append)
			subscription.Start()
		})
	}
	client.RepairEndpoint(&watch.Option{Endpoint: testWatchEndpoint(t, next), Username: watch.LinkUsername, Password: watch.LinkPassword})
	group.Wait()
	next.SetAndNotify("rpc:test:endpoint:demo", "live")
	for _, log := range logs {
		require.Eventually(t, log.has(watch.EventKindUpsert, "rpc:test:endpoint:demo", "live"), time.Second, time.Millisecond)
	}
}
