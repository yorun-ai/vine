package minder

import (
	"context"
	"time"

	"github.com/google/uuid"

	"go.yorun.ai/vine/internal/core/skel"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/util/goutil"
)

var heartbeatInterval = 10 * time.Second

func (i *AppInstance) startHeartbeat() {
	if i.minder.Flag.HubInprocMode {
		return
	}

	i.mutex.Lock()
	if i.heartbeatCancel != nil {
		i.mutex.Unlock()
		return
	}

	heartbeatCtx, cancel := context.WithCancel(i.minder.Context)
	i.heartbeatCancel = cancel
	i.mutex.Unlock()

	safeTicker := goutil.NewSafeTicker(heartbeatCtx, heartbeatInterval, nil)
	safeTicker.Go(func() {
		registered := i.minder.RegistryServiceClient.Heartbeat(hubskeled.AppStatus{
			Name:       i.AppInfo.Name(),
			InstanceId: skel.NewUUID(uuid.MustParse(i.AppInfo.InstanceId())),
		})
		if !registered {
			i.registerHubInstance()
		}
	})
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
