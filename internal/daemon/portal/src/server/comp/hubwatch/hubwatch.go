package hubwatch

import (
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/daemon"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type Client struct {
	hubapiwatch.Client

	Flag     *flag.Flag       `inject:""`
	HubInfo  *hubinfo.HubInfo `inject:""`
	Identity *mtls.Identity   `inject:""`
}

func (c *Client) InitOption(option *hubapiwatch.Option) {
	option.Username = hubapiwatch.PortalUsername
	option.Password = hubapiwatch.PortalPassword
	option.InprocMode = c.Flag.HubInprocMode
	if option.InprocMode {
		return
	}

	option.Endpoint = c.HubInfo.WatchEndpoint()
	if c.Identity.Enabled() {
		option.TLSConfig = c.Identity.ClientConfig(daemon.HubIdentity.SPIFFEPath())
	}
}
