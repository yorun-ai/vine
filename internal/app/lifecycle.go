package app

import (
	"context"
	"os/signal"
	"syscall"
)

// StartAndWait listens for termination before starting the application, then
// stops it after startup has completed and a signal has arrived.
func StartAndWait(application App) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application.Start()
	<-ctx.Done()
	stop()
	application.StopGracefully()
}
