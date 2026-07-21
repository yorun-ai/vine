package hubredis

import (
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type Client struct {
	hubapiredis.Client

	Flag    *flag.Flag       `inject:""`
	HubInfo *hubinfo.HubInfo `inject:""`
}

func (c *Client) InitOption(option *hubapiredis.Option) {
	option.InprocMode = c.Flag.HubInprocMode
	if option.InprocMode {
		return
	}

	option.Endpoint = c.HubInfo.RedisEndpoint()
}
