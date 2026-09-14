package repo

import (
	"testing"
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
)

func setTimeNowForTest(t *testing.T, now func() time.Time) {
	t.Helper()
	old := timeNow
	timeNow = now
	t.Cleanup(func() {
		timeNow = old
	})
	watchserver.SetTimeNowForTest(t, now)
}
