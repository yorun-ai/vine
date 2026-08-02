package hubredis

import (
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type Client struct {
	hubapiredis.Client

	Flag     *flag.Flag       `inject:""`
	HubInfo  *hubinfo.HubInfo `inject:""`
	Identity *mtls.Identity   `inject:""`
}

func (c *Client) InitOption(option *hubapiredis.Option) {
	option.Username = hubapiredis.PortalUsername
	option.Password = hubapiredis.PortalPassword
	option.InprocMode = c.Flag.HubInprocMode
	if option.InprocMode {
		return
	}

	option.Endpoint = c.HubInfo.RedisEndpoint()
	if c.Identity.Enabled() {
		option.TLSConfig = c.Identity.ClientConfig(daemon.HubIdentity.SPIFFEPath())
	}
}
