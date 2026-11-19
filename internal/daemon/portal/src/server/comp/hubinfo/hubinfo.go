package hubinfo

import (
	"context"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/daemon/hub/api/hubinfo"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/util/goutil"
)

// refreshInterval bounds how long Portal can keep using a stale Hub endpoint.
// Polling also repairs Portal when registration heartbeats cannot complete.
var refreshInterval = 30 * time.Second

var hubInfoLogger = logger.New("daemon:portal:hubinfo")

// HubInfo caches the endpoints Hub advertises at startup. A restarted Hub can
// advertise a different watch endpoint, so HubInfo refreshes the snapshot in the
// background and notifies listeners that own a connection.
type HubInfo struct {
	app.BaseComponent

	Context           context.Context             `inject:""`
	Flag              *flag.Flag                  `inject:""`
	InfoServiceClient hubskeled.InfoServiceClient `inject:""`

	host  string
	cache hubinfo.Cache

	mutex       sync.Mutex
	refreshStop context.CancelFunc
}

func (c *HubInfo) DIInit() {
	if c.Flag.HubInprocMode {
		return
	}

	c.host = c.Flag.HubEndpointURL.Hostname()
	c.cache.Init(c.InfoServiceClient.GetInfo())
}

func (c *HubInfo) BeforeAppStart() error {
	if c.Flag.HubInprocMode {
		return nil
	}

	refreshCtx, cancel := context.WithCancel(c.Context)

	c.mutex.Lock()
	c.refreshStop = cancel
	c.mutex.Unlock()

	safeTicker := goutil.NewSafeTicker(refreshCtx, refreshInterval, nil)
	safeTicker.Go(func() {
		goutil.RunWithRecover(func(recovered any) {
			hubInfoLogger.Warn("portal hub information refresh failed", "error", recovered)
		}, c.Refresh)
	})
	return nil
}

func (c *HubInfo) AfterAppStop() {
	c.mutex.Lock()
	cancel := c.refreshStop
	c.refreshStop = nil
	c.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
}

// OnRefresh registers a listener invoked after Refresh observed changed Hub
// information. Each listener decides whether an endpoint it owns moved.
func (c *HubInfo) OnRefresh(listener func()) {
	c.cache.OnRefresh(listener)
}

// Refresh re-reads Hub information and notifies listeners when it changed. An
// unreachable Hub panics like every other Hub call; background callers keep it
// inside their recovery scope and retry on the next interval.
func (c *HubInfo) Refresh() {
	if c.Flag.HubInprocMode {
		return
	}

	c.cache.Refresh(c.InfoServiceClient)
}

func (c *HubInfo) WatchEndpoint() string {
	return c.cache.WatchEndpoint(c.host)
}
