package embedded

import (
	"testing"
	"time"
)

func SetTimeNowForTest(t *testing.T, now func() time.Time) {
	t.Helper()

	old := timeNow
	timeNow = now
	t.Cleanup(func() {
		timeNow = old
	})
}
