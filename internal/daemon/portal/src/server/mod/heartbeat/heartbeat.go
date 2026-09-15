package heartbeat

import (
	"context"
	"time"
	"uuid"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/util/goutil"
)

// heartbeatInterval keeps the Portal instance lease alive. Hub expires the lease
// after 30 seconds, so a stopped Portal disappears from the Hub Dashboard even
// when it could not unregister itself.
var heartbeatInterval = 10 * time.Second

// portalStartedAt is the start of this Portal process, reported to Hub so the
// Dashboard can tell a recently restarted instance from a stable one.
var portalStartedAt = time.Now()

var heartbeatLogger = logger.New("daemon:portal:heartbeat")

// Heartbeat registers this Portal instance with Hub and keeps the registration
// alive, so Hub can report which Portals are serving. Standalone Portal shares
// Hub's process, so it registers without a heartbeat, like an application
// instance. Hub forgets Portal instances when it restarts, and the next
// heartbeat registers again.
type Heartbeat struct {
	app.BaseModule

	Context              context.Context                    `inject:""`
	Flag                 *flag.Flag                         `inject:""`
	App                  runtime.App                        `inject:""`
	HubInfo              *hubinfo.HubInfo                   `inject:""`
	PortalRegistryClient skeled.PortalRegistryServiceClient `inject:""`

	instanceId skel.UUID
	stop       context.CancelFunc
}

func (h *Heartbeat) AfterAppStart() {
	h.instanceId = skel.NewUUID(uuid.MustParse(h.App.InstanceId()))
	h.register()

	if h.Flag.HubInprocMode {
		// Standalone Portal lives in Hub's process, so its registration cannot
		// outlive Hub and needs no liveness refresh.
		return
	}

	heartbeatCtx, cancel := context.WithCancel(h.Context)
	h.stop = cancel
	goutil.NewSafeTicker(heartbeatCtx, heartbeatInterval, nil).Go(func() {
		if h.PortalRegistryClient.Heartbeat(skeled.PortalStatus{InstanceId: h.instanceId}) {
			return
		}
		// Hub no longer knows this instance, which is what a Hub restart looks
		// like. A restarted Hub can also advertise a new watch endpoint, so
		// refresh Hub information before registering again.
		goutil.RunWithRecover(heartbeatRecovered, h.HubInfo.Refresh)
		h.register()
	})
}

// BeforeAppStop unregisters while the application context is still live. A
// cancelled context cannot reach Hub, and Hub expires the lease on its own when
// this call cannot reach Hub.
func (h *Heartbeat) BeforeAppStop() {
	if h.stop != nil {
		h.stop()
		h.stop = nil
	}

	goutil.RunWithRecover(func(recovered any) {
		heartbeatLogger.Warn("portal unregister from Hub failed", "error", recovered)
	}, h.PortalRegistryClient.Unregister, h.instanceId)
}

func (h *Heartbeat) register() {
	// Best effort: a Hub without this service leaves the Portal running, and the
	// heartbeat registers again once Hub answers.
	goutil.RunWithRecover(func(recovered any) {
		heartbeatLogger.Warn("portal register with Hub failed", "error", recovered)
	}, func() {
		h.PortalRegistryClient.Register(skeled.PortalRegistration{
			InstanceId: h.instanceId,
			Version:    h.App.Version(),
			StartedAt:  skel.NewTimestamp(portalStartedAt),
		})
	})
}

func heartbeatRecovered(recovered any) {
	heartbeatLogger.Warn("portal hub information refresh failed", "error", recovered)
}
