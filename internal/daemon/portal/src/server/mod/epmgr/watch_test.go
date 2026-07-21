package epmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func TestManagerRpcWatchUsesRefCount(t *testing.T) {
	prefix := redised.FormatRpcServiceRegistrationPrefix("demo.UserService")
	manager := newTestManager(map[string]string{
		testRpcRegistrationKey("demo.UserService", "instance-1"): testRpcRegistrationValue("demo.UserService", "instance-1", "http://127.0.0.1:23001"),
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
	prefix := redised.FormatRpcServiceRegistrationPrefix("demo.UserService")
	manager := newTestManager(map[string]string{})
	watcher := manager.WatchRpc("demo.UserService")
	t.Cleanup(watcher.Release)

	key := testRpcRegistrationKey("demo.UserService", "instance-1")
	manager.handleRegistrationEvent(prefix, hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Key:   key,
		Value: testRpcRegistrationValue("demo.UserService", "instance-1", "http://127.0.0.1:23001"),
	})
	endpoint, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, endpoint)
	registration := endpoint.(*redised.RpcServiceRegistration)
	assert.Equal(t, "http://127.0.0.1:23001", registration.Endpoint)

	manager.handleRegistrationEvent(prefix, hubredis.Event{
		Kind: hubredis.EventKindDelete,
		Key:  key,
	})
	endpoint, configured = manager.nextEndpoint(prefix)
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func TestManagerWebWatchUsesRefCount(t *testing.T) {
	prefix := redised.FormatWebRegistrationPrefix("admin@demo.app")
	manager := newTestManager(map[string]string{
		testWebRegistrationKey("admin@demo.app", "instance-1"): testWebRegistrationValue("admin@demo.app", "instance-1", "http://127.0.0.1:23001"),
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
	prefix := redised.FormatWebRegistrationPrefix("admin@demo.app")
	manager := newTestManager(map[string]string{})
	watcher := manager.WatchWeb("admin@demo.app")
	t.Cleanup(watcher.Release)

	key := testWebRegistrationKey("admin@demo.app", "instance-1")
	manager.handleRegistrationEvent(prefix, hubredis.Event{
		Kind:  hubredis.EventKindUpsert,
		Key:   key,
		Value: testWebRegistrationValue("admin@demo.app", "instance-1", "http://127.0.0.1:23001"),
	})
	endpoint, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, endpoint)
	registration := endpoint.(*redised.WebRegistration)
	assert.Equal(t, "http://127.0.0.1:23001", registration.Endpoint)

	manager.handleRegistrationEvent(prefix, hubredis.Event{
		Kind: hubredis.EventKindDelete,
		Key:  key,
	})
	endpoint, configured = manager.nextEndpoint(prefix)
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}
