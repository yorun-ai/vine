package hubwatch

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type _RepairTestInfoServiceClient struct {
	info hubskeled.Info
}

func (c *_RepairTestInfoServiceClient) GetInfo(_ ...rpcclient.InvokeOption) hubskeled.Info {
	return c.info
}

func newRepairTestWatchServer(t *testing.T) *watchserver.Server {
	t.Helper()

	server := watchserver.NewServerForTest()
	server.Option.WatchListen = "127.0.0.1:0"
	server.DIInit()
	t.Cleanup(server.AfterAppStop)
	return server
}

func repairTestWatchPort(t *testing.T, server *watchserver.Server) int {
	t.Helper()

	addr := watchserver.WatchListenAddrForTest(t, server)
	require.NotEmpty(t, addr)
	_, portText, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return port
}

func TestHubInfoRefreshRepairsWatchEndpoint(t *testing.T) {
	previousHub := newRepairTestWatchServer(t)
	nextHub := newRepairTestWatchServer(t)
	ruleKey := watched.FormatPortalRuleKey("demo")
	previousHub.Set(ruleKey, "from-previous-hub")
	nextHub.Set(ruleKey, "from-next-hub")

	infoClient := &_RepairTestInfoServiceClient{info: hubskeled.Info{
		WatchPort: repairTestWatchPort(t, previousHub),
	}}
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize()
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

	value, ok := client.Load(ruleKey)
	require.True(t, ok)
	assert.Equal(t, "from-previous-hub", value)

	// A Hub restart that advertises a different watch port must move Portal's
	// subscription without a Portal restart, both for the poller and for the
	// registration heartbeat that refreshes Hub information.
	infoClient.info = hubskeled.Info{WatchPort: repairTestWatchPort(t, nextHub)}
	hubInfo.Refresh()

	value, ok = client.Load(ruleKey)
	require.True(t, ok)
	assert.Equal(t, "from-next-hub", value)
}
