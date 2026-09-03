//go:build goroutineleak

package scheduler

import (
	"testing"

	"go.yorun.ai/vine/internal/testutil/goroutineleak"
)

func TestGoroutineLeakSchedulerLifecycle(t *testing.T) {
	runSchedulerLifecycle()
	goroutineleak.RequireNone(t)
}

func runSchedulerLifecycle() {
	target := newTestScheduler(nil, &_SchedulerTaskPublisher{})
	target.AfterAppStart()
	target.BeforeAppStop()
}
