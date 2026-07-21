package hubinfo

import (
	"fmt"

	"go.yorun.ai/vine/internal/app"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type HubInfo struct {
	app.BaseComponent

	Flag              *flag.Flag                  `inject:""`
	InfoServiceClient hubskeled.InfoServiceClient `inject:""`

	host string
	info hubskeled.Info
}

func (c *HubInfo) DIInit() {
	if c.Flag.HubInprocMode {
		return
	}

	c.host = c.Flag.HubEndpointURL.Hostname()
	c.info = c.InfoServiceClient.GetInfo()
}

func (c *HubInfo) RedisEndpoint() string {
	return fmt.Sprintf("%s:%d", c.host, c.info.RedisPort)
}
