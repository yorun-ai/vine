//go:build goroutineleak

package app

import (
	"testing"

	"go.yorun.ai/vine/internal/testutil/goroutineleak"
)

func TestGoroutineLeakAppHTTPServerLifecycle(t *testing.T) {
	runAppHTTPServerLifecycle()
	goroutineleak.RequireNone(t)
}

func runAppHTTPServerLifecycle() {
	app := newTestAppImpl()
	app.startHTTPServer()
	app.stopHTTPServer()
}
