package hubinfo

import (
	"fmt"

	"go.yorun.ai/vine/internal/app"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
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

func (c *HubInfo) MQEndpoint() string {
	if c.info.NatsPort != 0 {
		scheme := "nats"
		if c.Flag.MTLS.Enabled() {
			scheme = "tls"
		}
		return fmt.Sprintf("%s://%s:%d", scheme, c.host, c.info.NatsPort)
	}
	return c.info.MqEndpoint
}

func (c *HubInfo) UsesEmbeddedNATS() bool {
	return c.info.NatsPort != 0
}
