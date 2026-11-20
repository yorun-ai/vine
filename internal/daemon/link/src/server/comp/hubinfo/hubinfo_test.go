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
	client := &_TestInfoServiceClient{info: hubskeled.Info{
		WatchPort:  7072,
		MqEmbedded: true,
		MqNatsPort: 4222,
	}}
	flags := &flag.Flag{HubEndpoint: "http://[fe80::1%25en0]:7071"}
	flags.Normalize(false)
	component := &HubInfo{Flag: flags, InfoServiceClient: client}

	component.DIInit()
	assert.Equal(t, 1, client.getInfoCall)
	assert.Equal(t, "[fe80::1%en0]:7072", component.WatchEndpoint())
	assert.Equal(t, "nats://[fe80::1%25en0]:4222", component.MQEndpoint())
}

func TestHubInfoRefreshNotifiesListenersOnlyWhenHubInfoChanged(t *testing.T) {
	client := &_TestInfoServiceClient{info: hubskeled.Info{
		MqEmbedded: true,
		MqNatsPort: 4222,
	}}
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize(false)
	component := &HubInfo{Flag: flags, InfoServiceClient: client}
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

func TestHubInfoRefreshRollsBackAndNotifiesEveryListenerAfterRepairFailure(t *testing.T) {
	client := &_TestInfoServiceClient{info: hubskeled.Info{WatchPort: 7072}}
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize(false)
	component := &HubInfo{Flag: flags, InfoServiceClient: client}
	component.DIInit()

	secondCalls := 0
	component.OnRefresh(func() { panic("repair failed") })
	component.OnRefresh(func() { secondCalls++ })
	client.info.WatchPort = 7073

	assert.PanicsWithValue(t, "repair failed", component.Refresh)
	assert.Equal(t, 1, secondCalls)
	assert.Equal(t, "127.0.0.1:7072", component.WatchEndpoint())

	assert.PanicsWithValue(t, "repair failed", component.Refresh)
	assert.Equal(t, 2, secondCalls)
}

func TestHubInfoInprocSkipsLookupOnInitAndRefresh(t *testing.T) {
	client := &_TestInfoServiceClient{}
	flags := &flag.Flag{
		HubInprocMode: true,
		HubEndpoint:   "rpc+inproc://vine/hub",
	}
	flags.Normalize(false)
	component := &HubInfo{Flag: flags, InfoServiceClient: client}
	component.DIInit()

	component.Refresh()

	assert.Equal(t, 0, client.getInfoCall)
}

func TestHubInfoUsesAdvertisedMQ(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payload  string
		endpoint string
		embedded bool
	}{
		{"embedded", `{"mqEmbedded":true,"mqNatsPort":4223}`, "nats://hub:4223", true},
		{"external", `{"mqNatsEndpoint":"nats://mq:4222"}`, "nats://mq:4222", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			component := &HubInfo{host: "hub", Flag: &flag.Flag{}}
			component.cache.Init(vcode.MustUnmarshalJsonS[hubskeled.Info](tc.payload))
			assert.Equal(t, tc.endpoint, component.MQEndpoint())
			assert.Equal(t, tc.embedded, component.UsesEmbeddedNATS())
		})
	}
}
