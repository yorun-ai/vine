package goroutineleak

import (
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
)

var profileMutex sync.Mutex

// RequireNone fails the test when the runtime detects a permanently blocked
// goroutine. Call it only after the lifecycle under test has returned so local
// references cannot hide otherwise unreachable synchronization primitives.
func RequireNone(t testing.TB) {
	t.Helper()

	profileMutex.Lock()
	defer profileMutex.Unlock()

	profile := pprof.Lookup("goroutineleak")
	if profile == nil {
		t.Fatal("runtime/pprof goroutineleak profile is unavailable")
	}

	var report strings.Builder
	if err := profile.WriteTo(&report, 1); err != nil {
		t.Fatalf("write goroutineleak profile: %v", err)
	}
	if count := profile.Count(); count != 0 {
		t.Fatalf("detected %d permanently blocked goroutine(s):\n%s", count, report.String())
	}
}
