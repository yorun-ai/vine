package watch_test

import (
	"context"
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

type _eventLog struct {
	mutex  sync.Mutex
	events []watch.Event
}

func (l *_eventLog) append(event watch.Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.events = append(l.events, event)
}

func (l *_eventLog) has(kind string, key string, value string) func() bool {
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

	events := new(_eventLog)
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
