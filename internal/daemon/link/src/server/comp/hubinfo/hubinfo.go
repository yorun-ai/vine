package hubinfo

import (
	"net"
	"net/url"
	"strconv"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/hubinfo"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

// HubInfo caches the endpoints and capabilities Hub advertises at startup.
// A restarted Hub can advertise different MQ or lock addresses, so background
// callers refresh this snapshot and listeners repair what they own.
type HubInfo struct {
	app.BaseComponent

	Flag              *flag.Flag                  `inject:""`
	InfoServiceClient hubskeled.InfoServiceClient `inject:""`

	host  string
	cache hubinfo.Cache
}

func (c *HubInfo) DIInit() {
	if c.Flag.HubInprocMode {
		return
	}

	c.host = c.Flag.HubEndpointURL.Hostname()
	c.cache.Init(c.InfoServiceClient.GetInfo())
}

// OnRefresh registers a listener invoked after Refresh observed changed Hub
// information. Each listener decides whether an endpoint it owns moved.
func (c *HubInfo) OnRefresh(listener func()) {
	c.cache.OnRefresh(listener)
}

// Refresh re-reads Hub information and notifies listeners when it changed.
// Unreachable Hub panics like every other Hub call, so background callers keep
// it inside their recovery scope and retry on the next attempt.
func (c *HubInfo) Refresh() {
	if c.Flag.HubInprocMode {
		return
	}

	c.cache.Refresh(c.InfoServiceClient)
}

func (c *HubInfo) WatchEndpoint() string {
	return c.cache.WatchEndpoint(c.host)
}

func (c *HubInfo) MQEndpoint() string {
	info := c.cache.Info()
	if usesEmbeddedNATS(info) {
		port := info.MqNatsPort
		if port == 0 {
			port = info.NatsPort
		}
		scheme := "nats"
		if c.Flag.MTLS.Enabled() {
			scheme = "tls"
		}
		endpoint := url.URL{Scheme: scheme, Host: net.JoinHostPort(c.host, strconv.Itoa(port))}
		return endpoint.String()
	}
	if info.MqNatsEndpoint != "" {
		return info.MqNatsEndpoint
	}
	return info.MqEndpoint
}

func usesEmbeddedNATS(info hubskeled.Info) bool {
	return info.MqEmbedded || info.NatsPort != 0
}

func (c *HubInfo) UsesEmbeddedNATS() bool {
	return usesEmbeddedNATS(c.cache.Info())
}

func (c *HubInfo) LockMode() string {
	if c.Flag.HubInprocMode {
		return hublock.ModeEmbedded
	}
	mode := c.cache.Info().LockMode
	if mode == "" {
		return hublock.ModeDisable
	}
	return mode
}

func (c *HubInfo) LockRedisEndpoint() string {
	return c.cache.Info().LockRedisEndpoint
}
