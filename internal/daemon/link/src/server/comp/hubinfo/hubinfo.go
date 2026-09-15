package hubinfo

import (
	"fmt"
	"sync"

	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"

	"go.yorun.ai/vine/internal/app"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	"go.yorun.ai/vine/util/vslice"
)

// HubInfo caches the endpoints and capabilities Hub advertises at startup.
// A restarted Hub can advertise different MQ or lock addresses, so background
// callers refresh this snapshot and listeners repair what they own.
type HubInfo struct {
	app.BaseComponent

	Flag              *flag.Flag                  `inject:""`
	InfoServiceClient hubskeled.InfoServiceClient `inject:""`

	mutex     sync.RWMutex
	host      string
	info      hubskeled.Info
	listeners []func()
}

func (c *HubInfo) DIInit() {
	if c.Flag.HubInprocMode {
		return
	}

	c.host = c.Flag.HubEndpointURL.Hostname()
	c.info = c.InfoServiceClient.GetInfo()
}

// OnRefresh registers a listener invoked after Refresh observed changed Hub
// information. Each listener decides whether an endpoint it owns moved.
func (c *HubInfo) OnRefresh(listener func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.listeners = append(c.listeners, listener)
}

// Refresh re-reads Hub information and notifies listeners when it changed.
// Unreachable Hub panics like every other Hub call, so background callers keep
// it inside their recovery scope and retry on the next attempt.
func (c *HubInfo) Refresh() {
	if c.Flag.HubInprocMode {
		return
	}

	info := c.InfoServiceClient.GetInfo()

	c.mutex.Lock()
	changed := c.info != info
	c.info = info
	listeners := vslice.Clone(c.listeners)
	c.mutex.Unlock()

	if !changed {
		return
	}
	for _, listener := range listeners {
		listener()
	}
}

func (c *HubInfo) WatchEndpoint() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return fmt.Sprintf("%s:%d", c.host, c.watchPort())
}

func (c *HubInfo) MQEndpoint() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.mqEndpoint()
}

func (c *HubInfo) UsesEmbeddedNATS() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.usesEmbeddedNATS()
}

func (c *HubInfo) LockMode() string {
	if c.Flag.HubInprocMode {
		return hublock.ModeEmbedded
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.info.LockMode == "" {
		return hublock.ModeDisable
	}
	return c.info.LockMode
}

func (c *HubInfo) LockRedisEndpoint() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.info.LockRedisEndpoint
}

func (c *HubInfo) watchPort() int {
	if c.info.WatchPort != 0 {
		return c.info.WatchPort
	}
	// Hubs predating watchPort only advertise redisPort.
	return c.info.RedisPort
}

func (c *HubInfo) usesEmbeddedNATS() bool {
	return c.info.MqEmbedded || c.info.NatsPort != 0
}

func (c *HubInfo) mqEndpoint() string {
	if c.usesEmbeddedNATS() {
		port := c.info.MqNatsPort
		if port == 0 {
			port = c.info.NatsPort
		}
		scheme := "nats"
		if c.Flag.MTLS.Enabled() {
			scheme = "tls"
		}
		return fmt.Sprintf("%s://%s:%d", scheme, c.host, port)
	}
	if c.info.MqNatsEndpoint != "" {
		return c.info.MqNatsEndpoint
	}
	return c.info.MqEndpoint
}
