//go:build goroutineleak

package goroutineleak

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

const knownLeakEnv = "VINE_TEST_KNOWN_GOROUTINE_LEAK"

func TestGoroutineLeakHelperDetectsKnownLeak(t *testing.T) {
	if os.Getenv(knownLeakEnv) == "1" {
		startKnownLeak()
		RequireNone(t)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=^TestGoroutineLeakHelperDetectsKnownLeak$")
	command.Env = append(os.Environ(), knownLeakEnv+"=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("known goroutine leak was not detected:\n%s", output)
	}
	if !bytes.Contains(output, []byte("permanently blocked goroutine")) {
		t.Fatalf("known goroutine leak failed without a leak report:\n%s", output)
	}

	RequireNone(t)
}

func startKnownLeak() {
	started := make(chan struct{})
	go func() {
		close(started)
		<-make(chan struct{})
	}()
	<-started
	for range 10 {
		runtime.Gosched()
	}
}
