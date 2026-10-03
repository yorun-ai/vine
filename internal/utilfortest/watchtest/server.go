package watchtest

import (
	"testing"

	"go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
)

type _ServerClient struct {
	watch.Client
	username string
}

func (c *_ServerClient) InitOption(option *watch.Option) {
	option.Username = c.username
	option.InprocMode = true
}

// NewServer starts a real Hub Watch server and an owned client over the in-process connection.
func NewServer(t *testing.T, username string) (*watchserver.Server, *watch.Client) {
	t.Helper()
	previous := watch.InprocServer()
	server := watchserver.NewServerForTest()
	server.InprocFlag.Enabled = true
	t.Cleanup(func() { server.AfterAppStop(); watch.SetInprocServer(previous) })
	server.DIInit()
	client := &_ServerClient{username: username}
	manager := &watch.ClientManager{Context: t.Context()}
	t.Cleanup(manager.AfterAppStop)
	manager.InitComponent(client)
	return server, &client.Client
}
