package epmgr

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
	"testing"
	"time"
	"uuid"
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
				testRpcRegistrationKey("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001")): testRpcRegistrationValue("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
				testRpcRegistrationKey("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000002")): testRpcRegistrationValue("demo.UserService", uuid.MustParse("00000000-0000-0000-0000-000000000002"), "http://127.0.0.1:23002"),
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
				testWebRegistrationKey("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001")): testWebRegistrationValue("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000001"), "http://127.0.0.1:23001"),
				testWebRegistrationKey("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000002")): testWebRegistrationValue("admin@demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000002"), "http://127.0.0.1:23002"),
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

	endpoint, _, configured := manager.NextRpcEndpoint("demo.UserService")
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

func testRpcRegistrationKey(serviceName string, instanceId uuid.UUID) string {
	return watched.FormatRpcServiceRegistrationKey(serviceName, "demo.app", instanceId)
}

func testRpcRegistrationValue(serviceName string, instanceId uuid.UUID, endpoint string) string {
	return vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
		Endpoint:      endpoint,
		ServiceName:   serviceName,
		AppName:       "demo.app",
		AppInstanceId: instanceId,
	})
}

func testWebRegistrationKey(webName string, instanceId uuid.UUID) string {
	return watched.FormatWebRegistrationKey(webName, "demo.app", instanceId)
}

func testWebRegistrationValue(webName string, instanceId uuid.UUID, endpoint string) string {
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
	key := testRpcRegistrationKey("demo.Service", uuid.MustParse("00000000-0000-0000-0000-00000000000a"))
	server.SetAndNotify(key, testRpcRegistrationValue("demo.Service", uuid.MustParse("00000000-0000-0000-0000-00000000000a"), "http://127.0.0.1:7082"))
	require.Eventually(t, func() bool {
		endpoint, _, _ := manager.NextRpcEndpoint("demo.Service")
		return endpoint != nil && endpoint.Endpoint == "http://127.0.0.1:7082"
	}, 2*time.Second, time.Millisecond)
	server.DeleteAndNotify(key)
	require.Eventually(t, func() bool { endpoint, _, _ := manager.NextRpcEndpoint("demo.Service"); return endpoint == nil }, 2*time.Second, time.Millisecond)
	_, _, configured := manager.NextRpcEndpoint("demo.Service")
	assert.True(t, configured)
	subscription.Release()
	_, _, configured = manager.NextRpcEndpoint("demo.Service")
	assert.False(t, configured)
}

type networkWatchClient struct {
	watch.Client
	endpoint string
}

func (c *networkWatchClient) InitOption(option *watch.Option) {
	option.Endpoint = c.endpoint
	option.Username = watch.PortalUsername
}

func TestPeriodicNetworkHubSnapshotRepairsMissedEndpointDelete(t *testing.T) {
	server := watchserver.NewServerForTest()
	server.Option.WatchListen = "127.0.0.1:0"
	server.DIInit()
	t.Cleanup(server.AfterAppStop)
	client := &networkWatchClient{endpoint: watchserver.WatchListenAddrForTest(t, server)}
	clientManager := &watch.ClientManager{Context: t.Context()}
	clientManager.InitComponent(client)
	t.Cleanup(clientManager.AfterAppStop)
	key := testRpcRegistrationKey(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a"))
	server.SetAndNotify(key, testRpcRegistrationValue(testService, uuid.MustParse("00000000-0000-0000-0000-00000000000a"), "http://old-ip"))
	manager := &Manager{Context: t.Context(), Watch: &client.Client}
	manager.DIInit()
	subscription := manager.WatchRpc(testService)
	t.Cleanup(subscription.Release)
	acquireAttempt(t, manager).Finish(nil)
	// Change the real Hub store without sending a deletion notification or
	// closing the subscription. The consumer must repair its retained cache.
	require.True(t, server.Del(key))
	acquireAttempt(t, manager).Finish(nil)
	require.Eventually(t, func() bool {
		registration, attempt, _ := manager.NextRpcEndpoint(testService)
		if attempt != nil {
			attempt.Finish(nil)
		}
		return registration == nil
	}, 40*time.Second, 10*time.Millisecond)
	_, _, configured := manager.NextRpcEndpoint(testService)
	require.True(t, configured)
}
