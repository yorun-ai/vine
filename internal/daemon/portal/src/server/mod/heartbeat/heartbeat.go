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
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
	"go.yorun.ai/vine/internal/util/goutil"
)

// heartbeatInterval keeps the Portal instance lease alive. Hub expires the lease
// after 30 seconds, so a stopped Portal disappears from the Hub Dashboard even
// when it could not unregister itself.
var heartbeatInterval = 10 * time.Second

var heartbeatLogger = logger.New("daemon:portal:heartbeat")

// Heartbeat registers this Portal instance with Hub and keeps the registration
// alive, so Hub can report which Portals are serving. Hub forgets Portal
// instances when it restarts, and the next heartbeat registers again.
type Heartbeat struct {
	app.BaseModule

	Context              context.Context                    `inject:""`
	Flag                 *flag.Flag                         `inject:""`
	App                  runtime.App                        `inject:""`
	PortalRegistryClient skeled.PortalRegistryServiceClient `inject:""`

	instanceId skel.UUID
	stop       context.CancelFunc
}

func (h *Heartbeat) AfterAppStart() {
	if h.Flag.HubInprocMode {
		// Hub and Portal share the process in inproc mode, where Hub keeps no
		// separate Portal liveness to report.
		return
	}

	h.instanceId = skel.NewUUID(uuid.MustParse(h.App.InstanceId()))
	h.register()

	heartbeatCtx, cancel := context.WithCancel(h.Context)
	h.stop = cancel
	goutil.NewSafeTicker(heartbeatCtx, heartbeatInterval, nil).Go(func() {
		if h.PortalRegistryClient.Heartbeat(skeled.PortalStatus{InstanceId: h.instanceId}) {
			return
		}
		// Hub no longer knows this instance, which is what a Hub restart looks
		// like, so register again.
		h.register()
	})
}

func (h *Heartbeat) AfterAppStop() {
	if h.stop != nil {
		h.stop()
		h.stop = nil
	}
	if h.Flag.HubInprocMode {
		return
	}

	// Best effort: Hub expires the lease on its own when this call cannot reach Hub.
	goutil.RunWithRecover(func(recovered any) {
		heartbeatLogger.Warn("portal unregister from Hub failed", "error", recovered)
	}, h.PortalRegistryClient.Unregister, h.instanceId)
}

func (h *Heartbeat) register() {
	h.PortalRegistryClient.Register(skeled.PortalRegistration{
		InstanceId: h.instanceId,
		Version:    h.App.Version(),
	})
}
