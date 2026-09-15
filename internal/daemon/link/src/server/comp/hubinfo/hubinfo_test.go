package hubinfo

import (
	"testing"

	"go.yorun.ai/vine/util/vcode"

	"github.com/stretchr/testify/assert"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _TestInfoServiceClient struct {
	info        hubskeled.Info
	getInfoCall int
}

func (c *_TestInfoServiceClient) GetInfo(_ ...rpcclient.InvokeOption) hubskeled.Info {
	c.getInfoCall++
	return c.info
}

func TestHubInfoDIInitLoadsHubInfo(t *testing.T) {
	client := &_TestInfoServiceClient{
		info: hubskeled.Info{
			WatchPort: 7072,
			NatsPort:  4222,
		},
	}
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize(false)
	component := &HubInfo{
		Flag:              flags,
		InfoServiceClient: client,
	}

	component.DIInit()
	assert.Equal(t, 1, client.getInfoCall)
	assert.Equal(t, "127.0.0.1:7072", component.WatchEndpoint())
	assert.Equal(t, "nats://127.0.0.1:4222", component.MQEndpoint())
}

func TestHubInfoRefreshNotifiesListenersOnlyWhenHubInfoChanged(t *testing.T) {
	client := &_TestInfoServiceClient{
		info: hubskeled.Info{
			MqEmbedded: true,
			MqNatsPort: 4222,
		},
	}
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize(false)
	component := &HubInfo{
		Flag:              flags,
		InfoServiceClient: client,
	}
	component.DIInit()

	refreshes := 0
	component.OnRefresh(func() { refreshes++ })

	component.Refresh()
	assert.Equal(t, 2, client.getInfoCall)
	assert.Equal(t, 0, refreshes)

	client.info.MqNatsPort = 4223
	component.Refresh()

	assert.Equal(t, 1, refreshes)
	assert.Equal(t, "nats://127.0.0.1:4223", component.MQEndpoint())
}

func TestHubInfoRefreshSkipsLookupInInprocMode(t *testing.T) {
	client := &_TestInfoServiceClient{}
	flags := &flag.Flag{
		HubInprocMode: true,
		HubEndpoint:   "rpc+inproc://vine/hub",
	}
	flags.Normalize(false)
	component := &HubInfo{
		Flag:              flags,
		InfoServiceClient: client,
	}
	component.DIInit()

	component.Refresh()

	assert.Equal(t, 0, client.getInfoCall)
}

func TestHubInfoDIInitSkipsHubInfoLookupInInprocMode(t *testing.T) {
	client := &_TestInfoServiceClient{}
	flags := &flag.Flag{
		HubInprocMode: true,
		HubEndpoint:   "rpc+inproc://vine/hub",
	}
	flags.Normalize(false)
	component := &HubInfo{
		Flag:              flags,
		InfoServiceClient: client,
	}

	component.DIInit()
	assert.Equal(t, 0, client.getInfoCall)
}

func TestHubInfoWatchEndpointSupportsHubVersions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    string
	}{
		{"old hub", `{"redisPort":7072}`, "hub:7072"},
		{"new hub", `{"watchPort":8072}`, "hub:8072"},
		{"prefer watch port", `{"watchPort":8072,"redisPort":7072}`, "hub:8072"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			component := &HubInfo{host: "hub", info: vcode.MustUnmarshalJsonS[hubskeled.Info](tc.payload)}
			assert.Equal(t, tc.want, component.WatchEndpoint())
		})
	}
}

func TestHubInfoMQSupportsHubVersions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payload  string
		endpoint string
		embedded bool
	}{
		{"old embedded", `{"natsPort":4222}`, "nats://hub:4222", true},
		{"new embedded", `{"mqEmbedded":true,"mqNatsPort":4223}`, "nats://hub:4223", true},
		{"prefer new port", `{"mqEmbedded":true,"mqNatsPort":4223,"natsPort":4222}`, "nats://hub:4223", true},
		{"old external", `{"mqEndpoint":"nats://old:4222"}`, "nats://old:4222", false},
		{"new external", `{"mqNatsEndpoint":"nats://new:4222"}`, "nats://new:4222", false},
		{"prefer new endpoint", `{"mqNatsEndpoint":"nats://new:4222","mqEndpoint":"nats://old:4222"}`, "nats://new:4222", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			component := &HubInfo{host: "hub", Flag: &flag.Flag{}, info: vcode.MustUnmarshalJsonS[hubskeled.Info](tc.payload)}
			assert.Equal(t, tc.endpoint, component.MQEndpoint())
			assert.Equal(t, tc.embedded, component.UsesEmbeddedNATS())
		})
	}
}
