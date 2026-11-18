package epmgr

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
)

type capturedEndpointWatch struct {
	*watchtest.Client
	handle func(hubwatch.Event)
}

func (c *capturedEndpointWatch) LoadListAndSubscribe(ctx context.Context, prefix string, handle func(hubwatch.Event)) (map[string]string, hubwatch.Subscription) {
	c.handle = handle
	return c.Client.LoadListAndSubscribe(ctx, prefix, handle)
}

func TestReleasedWatchCannotChangeReplacementEndpoints(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	key := testRpcRegistrationKey(testService, id)
	client := &capturedEndpointWatch{Client: watchtest.New(t, nil)}
	manager := &Manager{Context: t.Context(), Watch: client}
	manager.DIInit()
	old := manager.WatchRpc(testService)
	lateEvent := client.handle
	old.Release()
	client.SetValue(key, testRpcRegistrationValue(testService, id, "http://new"))
	current := manager.WatchRpc(testService)
	defer current.Release()

	lateEvent(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: key, Value: testRpcRegistrationValue(testService, id, "http://old")})
	lateEvent(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	registration, attempt, configured := manager.NextRpcEndpoint(testService)
	require.True(t, configured)
	require.NotNil(t, registration)
	require.Equal(t, "http://new", registration.Endpoint)
	attempt.Finish(nil)
	client.handle(hubwatch.Event{Kind: hubwatch.EventKindDelete, Key: key})
	registration, _, configured = manager.NextRpcEndpoint(testService)
	require.True(t, configured)
	require.Nil(t, registration)
}

func TestManagerRpcWatchUsesRefCount(t *testing.T) {
	prefix := watched.FormatRpcServiceRegistrationPrefix("demo.UserService")
	manager := newTestManager(t, map[string]string{
		testRpcRegistrationKey("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001")): testRpcRegistrationValue("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
	})

	watcher1 := manager.WatchRpc("demo.UserService")
	watcher2 := manager.WatchRpc("demo.UserService")

	route := manager.routesByPrefix[prefix]
	require.NotNil(t, route)
	assert.Equal(t, 2, route.refCount)

	watcher1.Release()
	_, configured := manager.nextEndpoint(prefix)
	assert.True(t, configured)
	assert.Equal(t, 1, route.refCount)

	watcher1.Release()
	assert.Equal(t, 1, route.refCount)

	watcher2.Release()
	_, configured = manager.nextEndpoint(prefix)
	assert.False(t, configured)
}

func TestManagerHandlesRpcRegistrationEvents(t *testing.T) {
	manager := newTestManager(t, map[string]string{})
	watcher := manager.WatchRpc("demo.UserService")
	t.Cleanup(watcher.Release)

	key := testRpcRegistrationKey("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	manager.handleRegistrationEvent(watcher.route, hubwatch.Event{
		Kind:  hubwatch.EventKindUpsert,
		Key:   key,
		Value: testRpcRegistrationValue("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
	})
	endpoint, configured := manager.nextEndpoint(watcher.route.prefix)
	require.True(t, configured)
	require.NotNil(t, endpoint)
	registration := endpoint.(*watched.RpcServiceRegistration)
	assert.Equal(t, "http://127.0.0.1:23001", registration.Endpoint)

	manager.handleRegistrationEvent(watcher.route, hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	endpoint, configured = manager.nextEndpoint(watcher.route.prefix)
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func TestManagerWebWatchUsesRefCount(t *testing.T) {
	prefix := watched.FormatWebRegistrationPrefix("admin@demo.app")
	manager := newTestManager(t, map[string]string{
		testWebRegistrationKey("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001")): testWebRegistrationValue("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
	})

	watcher1 := manager.WatchWeb("admin@demo.app")
	watcher2 := manager.WatchWeb("admin@demo.app")

	route := manager.routesByPrefix[prefix]
	require.NotNil(t, route)
	assert.Equal(t, 2, route.refCount)

	watcher1.Release()
	_, configured := manager.nextEndpoint(prefix)
	assert.True(t, configured)
	assert.Equal(t, 1, route.refCount)

	watcher1.Release()
	assert.Equal(t, 1, route.refCount)

	watcher2.Release()
	_, configured = manager.nextEndpoint(prefix)
	assert.False(t, configured)
}

func TestManagerHandlesWebRegistrationEvents(t *testing.T) {
	manager := newTestManager(t, map[string]string{})
	watcher := manager.WatchWeb("admin@demo.app")
	t.Cleanup(watcher.Release)

	key := testWebRegistrationKey("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	manager.handleRegistrationEvent(watcher.route, hubwatch.Event{
		Kind:  hubwatch.EventKindUpsert,
		Key:   key,
		Value: testWebRegistrationValue("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
	})
	endpoint, configured := manager.nextEndpoint(watcher.route.prefix)
	require.True(t, configured)
	require.NotNil(t, endpoint)
	registration := endpoint.(*watched.WebRegistration)
	assert.Equal(t, "http://127.0.0.1:23001", registration.Endpoint)

	manager.handleRegistrationEvent(watcher.route, hubwatch.Event{
		Kind: hubwatch.EventKindDelete,
		Key:  key,
	})
	endpoint, configured = manager.nextEndpoint(watcher.route.prefix)
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}
