package app

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

type signalLifecycleApp struct {
	t       *testing.T
	signal  syscall.Signal
	started bool
	stopped bool
}

func (*signalLifecycleApp) Name() string { return "signal-lifecycle-test" }

func (a *signalLifecycleApp) Start() {
	// Observe delivery before returning from Start, so this exercises a signal
	// received during startup rather than relying on scheduler timing.
	received := make(chan os.Signal, 1)
	signal.Notify(received, a.signal)
	defer signal.Stop(received)
	if err := syscall.Kill(os.Getpid(), a.signal); err != nil {
		a.t.Fatal(err)
	}
	<-received
	if a.stopped {
		a.t.Fatal("application stopped before startup completed")
	}
	a.started = true
}

func (a *signalLifecycleApp) StopGracefully() {
	if !a.started || a.stopped {
		a.t.Fatal("shutdown must occur once after startup completes")
	}
	a.stopped = true
}

func (a *signalLifecycleApp) StartAndWait() { StartAndWait(a) }

func TestStartAndWaitSignalDuringStartup(t *testing.T) {
	if childSignal := os.Getenv("VINE_TEST_STARTUP_SIGNAL"); childSignal != "" {
		sig := syscall.SIGTERM
		if childSignal == "interrupt" {
			sig = syscall.SIGINT
		}
		application := &signalLifecycleApp{t: t, signal: sig}
		StartAndWait(application)
		if !application.stopped {
			t.Fatal("application did not stop")
		}
		return
	}

	// Real process signals stay isolated from the test runner and other tests.
	for _, name := range []string{"terminate", "interrupt"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStartAndWaitSignalDuringStartup$", "-test.count=1")
			cmd.Env = append(os.Environ(), "VINE_TEST_STARTUP_SIGNAL="+name)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("startup signal lifecycle failed: %v\n%s", err, output)
			}
		})
	}
}
