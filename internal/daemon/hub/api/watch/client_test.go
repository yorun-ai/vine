package watch_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
)

type portalTestClient struct {
	watch.Client
	endpoint string
}

func (c *portalTestClient) InitOption(option *watch.Option) {
	option.Endpoint = c.endpoint
	option.Username = watch.PortalUsername
	option.Password = watch.PortalPassword
}

func TestLoadListReadsDescriptorSnapshotWithoutSubscription(t *testing.T) {
	server := newTestWatchServer(t)
	server.SetAndNotify("descriptor:data:demo.User", "user")
	server.SetAndNotify("descriptor:enum:demo.Status", "status")
	client := &portalTestClient{endpoint: testWatchEndpoint(t, server)}
	manager := &watch.ClientManager{Context: context.Background()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)
	require.Equal(t, map[string]string{"descriptor:data:demo.User": "user", "descriptor:enum:demo.Status": "status"}, client.LoadList("descriptor"))
	require.Equal(t, map[string]string{"descriptor:data:demo.User": "user"}, client.LoadList("descriptor:data"))
}
