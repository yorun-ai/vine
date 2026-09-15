package hubwatch

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func newTestWatchServer(t *testing.T) *watchserver.Server {
	t.Helper()

	server := watchserver.NewServerForTest()
	server.Option.WatchListen = "127.0.0.1:0"
	server.DIInit()
	t.Cleanup(server.AfterAppStop)
	return server
}

func TestHubInfoRefreshRepairsWatchEndpoint(t *testing.T) {
	previousHub := newTestWatchServer(t)
	nextHub := newTestWatchServer(t)
	previousHub.Set("config:demo", "from-previous-hub")
	nextHub.Set("config:demo", "from-next-hub")

	infoClient := &_TestInfoServiceClient{info: hubskeled.Info{
		WatchPort: hubPort(t, previousHub),
	}}
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize(false)
	hubInfo := &hubinfo.HubInfo{Flag: flags, InfoServiceClient: infoClient}
	hubInfo.DIInit()

	client := &Client{
		Flag:     flags,
		HubInfo:  hubInfo,
		Identity: mtls.DisabledIdentity(),
	}
	client.DIInit()
	manager := &hubapiwatch.ClientManager{Context: t.Context()}
	manager.InitComponent(client)
	t.Cleanup(manager.AfterAppStop)

	value, ok := client.Load("config:demo")
	require.True(t, ok)
	assert.Equal(t, "from-previous-hub", value)

	// A Hub restart that advertises a different watch port must move the client
	// without a Link restart.
	infoClient.info = hubskeled.Info{WatchPort: hubPort(t, nextHub)}
	hubInfo.Refresh()

	value, ok = client.Load("config:demo")
	require.True(t, ok)
	assert.Equal(t, "from-next-hub", value)
}

func hubPort(t *testing.T, server *watchserver.Server) int {
	t.Helper()

	addr := watchserver.WatchListenAddrForTest(t, server)
	require.NotEmpty(t, addr)
	_, portText, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return port
}
