package minder

import (
	"context"
	"time"

	appskeled "go.yorun.ai/vine/internal/core/app/skeled"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/runtime"
	"go.yorun.ai/vine/internal/util/goutil"
)

var (
	healthcheckInterval       = 5 * time.Second
	healthcheckPingTimeout    = 2 * time.Second
	healthcheckMaxFailedCount = 3
)

// newConsoleServiceClient is replaced in tests to stub app console health checks.
var newConsoleServiceClient = func(ctx context.Context, app runtime.App, endpoint string) appskeled.ConsoleServiceClientER {
	trace := meta.InitialTrace()
	actor := meta.NewAbsentActor()
	rpcCtx := meta.NewContext(ctx, trace, nil, actor)
	return appskeled.NewConsoleServiceClientER(rpcclient.New(rpcclient.Option{
		Context:        rpcCtx,
		ClientApp:      app,
		Logger:         logger.NewGlobalLogger(),
		ServerEndpoint: endpoint,
	}))
}

func (i *AppInstance) startHealthcheck(onFailure func()) {
	if i.minder.InprocFlag.Enabled {
		return
	}

	i.mutex.Lock()
	if i.healthcheckCancel != nil {
		i.mutex.Unlock()
		return
	}
	checkCtx, cancel := context.WithCancel(i.minder.Context)
	i.healthcheckCancel = cancel
	i.mutex.Unlock()

	failedCount := 0
	consoleEndpoint := i.ConsoleEndpoint
	consoleServiceClient := newConsoleServiceClient(checkCtx, i.minder.App, consoleEndpoint)
	safeTicker := goutil.NewSafeTicker(checkCtx, healthcheckInterval, nil)
	safeTicker.Go(func() {
		err := consoleServiceClient.Ping(rpcclient.WithTimeout(healthcheckPingTimeout))
		if err != nil && err.Code() == ex.InvocationTimeout {
			logger.Warn("minder console ping timed out",
				"instanceId", i.AppInfo.InstanceId(),
				"endpoint", consoleEndpoint,
				"timeout", healthcheckPingTimeout,
			)
			return
		}
		if err == nil {
			if failedCount > 0 {
				logger.Info("minder console ping recovered",
					"instanceId", i.AppInfo.InstanceId(),
					"endpoint", consoleEndpoint,
					"failedCount", failedCount,
				)
			}
			failedCount = 0
			return
		}

		failedCount++
		logger.Warn("minder console ping failed",
			"instanceId", i.AppInfo.InstanceId(),
			"endpoint", consoleEndpoint,
			"failedCount", failedCount,
			"maxFailedCount", healthcheckMaxFailedCount,
			"error", err,
		)
		if failedCount < healthcheckMaxFailedCount {
			return
		}

		logger.Error("minder unregistering instance after repeated health check failures",
			"instanceId", i.AppInfo.InstanceId(),
			"endpoint", consoleEndpoint,
			"failedCount", failedCount,
			"error", err,
		)
		i.stopHealthcheck()
		onFailure()
	})
}

func (i *AppInstance) stopHealthcheck() {
	if i.minder.InprocFlag.Enabled {
		return
	}

	i.mutex.Lock()
	cancel := i.healthcheckCancel
	i.healthcheckCancel = nil
	i.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
}
