package epmgr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/util/vcode"
)

func TestManagerRpcNextEndpointRoundRobins(t *testing.T) {
	prefix := watched.FormatRpcServiceRegistrationPrefix("demo.UserService")
	manager := newTestManager(map[string]string{
		testRpcRegistrationKey("demo.UserService", "instance-1"): testRpcRegistrationValue("demo.UserService", "instance-1", "http://127.0.0.1:23001"),
		testRpcRegistrationKey("demo.UserService", "instance-2"): testRpcRegistrationValue("demo.UserService", "instance-2", "http://127.0.0.1:23002"),
	})
	watcher := manager.WatchRpc("demo.UserService")
	t.Cleanup(watcher.Release)

	first, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, first)
	second, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, second)
	third, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, third)

	assert.NotEqual(t, first.(*watched.RpcServiceRegistration).Endpoint, second.(*watched.RpcServiceRegistration).Endpoint)
	assert.Equal(t, first.(*watched.RpcServiceRegistration).Endpoint, third.(*watched.RpcServiceRegistration).Endpoint)
}

func TestManagerNextRpcEndpointReturnsConfiguredWithoutEndpoint(t *testing.T) {
	manager := newTestManager(map[string]string{})
	watcher := manager.WatchRpc("demo.UserService")
	t.Cleanup(watcher.Release)

	endpoint, configured := manager.NextRpcEndpoint("demo.UserService")
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func TestManagerWebNextEndpointRoundRobins(t *testing.T) {
	prefix := watched.FormatWebRegistrationPrefix("admin@demo.app")
	manager := newTestManager(map[string]string{
		testWebRegistrationKey("admin@demo.app", "instance-1"): testWebRegistrationValue("admin@demo.app", "instance-1", "http://127.0.0.1:23001"),
		testWebRegistrationKey("admin@demo.app", "instance-2"): testWebRegistrationValue("admin@demo.app", "instance-2", "http://127.0.0.1:23002"),
	})
	watcher := manager.WatchWeb("admin@demo.app")
	t.Cleanup(watcher.Release)

	first, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, first)
	second, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, second)
	third, configured := manager.nextEndpoint(prefix)
	require.True(t, configured)
	require.NotNil(t, third)

	assert.NotEqual(t, first.(*watched.WebRegistration).Endpoint, second.(*watched.WebRegistration).Endpoint)
	assert.Equal(t, first.(*watched.WebRegistration).Endpoint, third.(*watched.WebRegistration).Endpoint)
}

func TestManagerNextWebEndpointReturnsConfiguredWithoutEndpoint(t *testing.T) {
	manager := newTestManager(map[string]string{})
	watcher := manager.WatchWeb("admin@demo.app")
	t.Cleanup(watcher.Release)

	endpoint, configured := manager.NextWebEndpoint("admin@demo.app")
	assert.True(t, configured)
	assert.Nil(t, endpoint)
}

func newTestManager(valuesByKey map[string]string) *Manager {
	manager := &Manager{
		Context: context.Background(),
		Watch:   hubwatch.NewTestClient(valuesByKey),
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
