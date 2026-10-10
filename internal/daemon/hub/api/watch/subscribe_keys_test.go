package watch_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

func TestKeySetSubscriptionBuffersEventsAndRepairsAsOneWatcher(t *testing.T) {
	previous := newTestWatchServer(t)
	next := newTestWatchServer(t)
	dataKey := "descriptor:data:demo.User"
	enumKey := "descriptor:enum:demo.Status"
	previous.SetAndNotify(dataKey, "old")
	previous.SetAndNotify(enumKey, "old-enum")
	client := &portalTestClient{endpoint: testWatchEndpoint(t, previous)}
	manager := &watch.ClientManager{Context: context.Background()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	events := new(_EventLog)
	values, subscription, err := client.LoadKeysAndSubscribe(ctx, []string{dataKey, enumKey}, events.append)
	require.NoError(t, err)
	require.Equal(t, map[string]string{dataKey: "old", enumKey: "old-enum"}, values)
	previous.SetAndNotify(dataKey, "new")
	previous.DeleteAndNotify(enumKey)
	previous.SetAndNotify("descriptor:data:demo.Unused", "ignored")
	subscription.Start()
	require.Eventually(t, events.has(watch.EventKindUpsert, dataKey, "new"), 5*time.Second, time.Millisecond)
	require.Eventually(t, events.has(watch.EventKindDelete, enumKey, ""), 5*time.Second, time.Millisecond)
	next.SetAndNotify(enumKey, "restored")
	client.endpoint = testWatchEndpoint(t, next)
	option := new(watch.Option)
	client.InitOption(option)
	require.True(t, client.RepairEndpoint(option))
	require.Eventually(t, events.has(watch.EventKindDelete, dataKey, ""), 5*time.Second, time.Millisecond)
	require.Eventually(t, events.has(watch.EventKindUpsert, enumKey, "restored"), 5*time.Second, time.Millisecond)
	events.mutex.Lock()
	defer events.mutex.Unlock()
	for _, event := range events.events {
		require.Contains(t, []string{dataKey, enumKey}, event.Key)
	}
}
