package hubredis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _TestInfoServiceClient struct {
	info hubskeled.Info
}

func (c *_TestInfoServiceClient) GetInfo(_ ...rpcclient.InvokeOption) hubskeled.Info {
	return c.info
}

func TestClientInitOptionUsesHubInfoRedisEndpoint(t *testing.T) {
	flags := &flag.Flag{
		HubEndpoint: "http://127.0.0.1:7071",
	}
	flags.Normalize(false)
	infoComponent := &hubinfo.HubInfo{
		Flag: flags,
		InfoServiceClient: &_TestInfoServiceClient{
			info: hubskeled.Info{
				RedisPort: 7073,
			},
		},
	}
	infoComponent.DIInit()

	client := &Client{
		Flag:    flags,
		HubInfo: infoComponent,
	}

	option := &hubapiredis.Option{}
	client.InitOption(option)

	assert.False(t, option.InprocMode)
	assert.Equal(t, "127.0.0.1:7073", option.Endpoint)
}

func TestClientInitOptionSkipsHubInfoInInprocMode(t *testing.T) {
	flags := &flag.Flag{
		HubInprocMode: true,
		HubEndpoint:   "rpc+inproc://vine/hub",
	}
	flags.Normalize(false)
	client := &Client{
		Flag:    flags,
		HubInfo: &hubinfo.HubInfo{},
	}

	option := &hubapiredis.Option{}
	client.InitOption(option)

	assert.True(t, option.InprocMode)
	assert.Empty(t, option.Endpoint)
}
