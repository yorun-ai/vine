package minder

import (
	"context"
	"time"

	"go.yorun.ai/vine/internal/core/logger"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/util/goutil"
)

var heartbeatInterval = 10 * time.Second

var heartbeatLogger = logger.New("daemon:link:minder:heartbeat")

func (i *AppInstance) startHeartbeat() {
	if i.minder.Flag.HubInprocMode {
		return
	}

	i.mutex.Lock()
	if i.stopping || i.heartbeatCancel != nil {
		i.mutex.Unlock()
		return
	}

	heartbeatCtx, cancel := context.WithCancel(i.minder.Context)
	i.heartbeatCancel = cancel
	i.mutex.Unlock()

	safeTicker := goutil.NewSafeTicker(heartbeatCtx, heartbeatInterval, nil)
	safeTicker.Go(func() {
		if i.hubKnowsInstance() || i.isStopping() {
			return
		}
		// Hub lost this instance, which is what a Hub restart looks like. A
		// restarted Hub can also advertise new MQ and lock endpoints, so refresh
		// Hub information before registering again.
		refreshed := false
		goutil.RunWithRecover(heartbeatRecovered, func() {
			i.minder.HubInfo.Refresh()
			refreshed = true
		})
		if !refreshed {
			return
		}
		goutil.RunWithRecover(heartbeatRecovered, i.registerHubInstance)
	})
}

// hubKnowsInstance reports whether Hub still tracks this instance. An
// unreachable Hub counts as a miss so the caller repairs and registers again.
// Stopping instances skip renewal and suppress recovery.
func (i *AppInstance) hubKnowsInstance() bool {
	i.hubMutex.Lock()
	defer i.hubMutex.Unlock()
	if i.isStopping() {
		return true
	}
	registered := false
	goutil.RunWithRecover(func(recovered any) {
		heartbeatLogger.Warn("minder hub heartbeat failed", "error", recovered)
	}, func() {
		registered = i.minder.RegistryServiceClient.Heartbeat(hubskeled.AppStatus{
			Name:       i.AppInfo.Name(),
			InstanceId: i.AppInfo.InstanceId(),
		}, rpcclient.WithTimeout(hubRegistryTimeout))
	})
	return registered
}

func heartbeatRecovered(recovered any) {
	heartbeatLogger.Warn("minder hub recovery step failed", "error", recovered)
}

func (i *AppInstance) stopHeartbeat() {
	if i.minder.Flag.HubInprocMode {
		return
	}

	i.mutex.Lock()
	cancel := i.heartbeatCancel
	i.heartbeatCancel = nil
	i.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
}
