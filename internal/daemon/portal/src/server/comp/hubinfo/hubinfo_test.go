package hubinfo

import (
	"testing"

	"go.yorun.ai/vine/util/vcode"

	"github.com/stretchr/testify/assert"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
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
		},
	}
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize()
	component := &HubInfo{
		Flag:              flags,
		InfoServiceClient: client,
	}

	component.DIInit()
	assert.Equal(t, 1, client.getInfoCall)
	assert.Equal(t, "127.0.0.1:7072", component.WatchEndpoint())
}

func TestHubInfoDIInitSkipsHubInfoLookupInInprocMode(t *testing.T) {
	client := &_TestInfoServiceClient{}
	flags := &flag.Flag{
		HubInprocMode: true,
	}
	flags.Normalize()
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
