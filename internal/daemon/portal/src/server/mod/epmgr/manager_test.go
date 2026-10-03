package epmgr

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
	"testing"
	"time"
)

func TestManagerNextEndpointRoundRobins(t *testing.T) {
	for _, test := range []struct {
		name        string
		prefix      string
		valuesByKey map[string]string
		watch       func(manager *Manager) *Watcher
		endpointOf  func(value any) string
	}{
		{
			name:   "rpc registration",
			prefix: watched.FormatRpcServiceRegistrationPrefix("demo.UserService"),
			valuesByKey: map[string]string{
				testRpcRegistrationKey("demo.UserService", "instance-1"): testRpcRegistrationValue("demo.UserService", "instance-1", "http://127.0.0.1:23001"),
				testRpcRegistrationKey("demo.UserService", "instance-2"): testRpcRegistrationValue("demo.UserService", "instance-2", "http://127.0.0.1:23002"),
			},
			watch: func(manager *Manager) *Watcher {
				return manager.WatchRpc("demo.UserService")
			},
			endpointOf: func(value any) string {
				return value.(*watched.RpcServiceRegistration).Endpoint
			},
		},
		{
			name:   "web registration",
			prefix: watched.FormatWebRegistrationPrefix("admin@demo.app"),
			valuesByKey: map[string]string{
				testWebRegistrationKey("admin@demo.app", "instance-1"): testWebRegistrationValue("admin@demo.app", "instance-1", "http://127.0.0.1:23001"),
				testWebRegistrationKey("admin@demo.app", "instance-2"): testWebRegistrationValue("admin@demo.app", "instance-2", "http://127.0.0.1:23002"),
			},
			watch: func(manager *Manager) *Watcher {
				return manager.WatchWeb("admin@demo.app")
			},
			endpointOf: func(value any) string {
				return value.(*watched.WebRegistration).Endpoint
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := newTestManager(t, test.valuesByKey)
			watcher := test.watch(manager)
			t.Cleanup(watcher.Release)

			first, configured := manager.nextEndpoint(test.prefix)
			require.True(t, configured)
			require.NotNil(t, first)
			second, configured := manager.nextEndpoint(test.prefix)
			require.True(t, configured)
			require.NotNil(t, second)
			third, configured := manager.nextEndpoint(test.prefix)
			require.True(t, configured)
			require.NotNil(t, third)

			assert.NotEqual(t, test.endpointOf(first), test.endpointOf(second))
			assert.Equal(t, test.endpointOf(first), test.endpointOf(third))
		})
	}
}

func TestManagerNextRpcEndpointReturnsConfiguredWithoutEndpoint(t *testing.T) {
	manager := newTestManager(t, map[string]string{})
	watcher := manager.WatchRpc("demo.UserService")
	t.Cleanup(watcher.Release)

	endpoint, configured := manager.NextRpcEndpoint("demo.UserService")
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func TestManagerNextWebEndpointReturnsConfiguredWithoutEndpoint(t *testing.T) {
	manager := newTestManager(t, map[string]string{})
	watcher := manager.WatchWeb("admin@demo.app")
	t.Cleanup(watcher.Release)

	endpoint, configured := manager.NextWebEndpoint("admin@demo.app")
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func newTestManager(t *testing.T, valuesByKey map[string]string) *Manager {
	manager := &Manager{
		Context: context.Background(),
		Watch:   watchtest.New(t, valuesByKey),
	}
	manager.DIInit()
	return manager
}

func testRpcRegistrationKey(serviceName string, instanceId string) string {
	return watched.FormatRpcServiceRegistrationKey(serviceName, "demo.app", instanceId)
}

func testRpcRegistrationValue(serviceName string, instanceId string, endpoint string) string {
	return vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
		Endpoint:      endpoint,
		ServiceName:   serviceName,
		AppName:       "demo.app",
		AppInstanceId: instanceId,
	})
}

func testWebRegistrationKey(webName string, instanceId string) string {
	return watched.FormatWebRegistrationKey(webName, "demo.app", instanceId)
}

func testWebRegistrationValue(webName string, instanceId string, endpoint string) string {
	return vcode.MustMarshalJsonS(watched.WebRegistration{
		Endpoint:      endpoint,
		WebSkelName:   webName,
		AppName:       "demo.app",
		AppInstanceId: instanceId,
	})
}

func TestEndpointDiscoveryThroughHubWatch(t *testing.T) {
	server, client := watchtest.NewServer(t, watch.PortalUsername)
	manager := &Manager{Context: t.Context(), Watch: client}
	manager.DIInit()
	subscription := manager.WatchRpc("demo.Service")
	t.Cleanup(subscription.Release)
	key := testRpcRegistrationKey("demo.Service", "instance")
	server.SetAndNotify(key, testRpcRegistrationValue("demo.Service", "instance", "http://127.0.0.1:7082"))
	require.Eventually(t, func() bool {
		endpoint, _ := manager.NextRpcEndpoint("demo.Service")
		return endpoint != nil && endpoint.Endpoint == "http://127.0.0.1:7082"
	}, 2*time.Second, time.Millisecond)
	server.DeleteAndNotify(key)
	require.Eventually(t, func() bool { endpoint, _ := manager.NextRpcEndpoint("demo.Service"); return endpoint == nil }, 2*time.Second, time.Millisecond)
	_, configured := manager.NextRpcEndpoint("demo.Service")
	assert.True(t, configured)
	subscription.Release()
	_, configured = manager.NextRpcEndpoint("demo.Service")
	assert.False(t, configured)
}
