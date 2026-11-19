package goutil

import (
	"os"
	"os/exec"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
)

func TestGoSafelyExecutesFunction(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		done := make(chan int, 1)

		GoSafely(func(a int, b int) {
			done <- a + b
		}, 2, 5)
		synctest.Wait()

		assert.Equal(t, 7, <-done)
	})
}

func TestGoSafelyRecoversPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ran := false
		GoSafely(func() {
			ran = true
			panic("boom")
		})
		synctest.Wait()
		assert.True(t, ran)
	})
}

func TestRescueRecoversPanic(t *testing.T) {
	const childEnv = "GO_WANT_GOUTIL_RESCUE_HELPER"
	if os.Getenv(childEnv) == "1" {
		defer Rescue()
		panic("boom")
	}

	command := exec.Command(os.Args[0], "-test.run=^TestRescueRecoversPanic$")
	command.Env = append(os.Environ(), childEnv+"=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("child process failed: %v\n%s", err, output)
	}
	assert.Contains(t, string(output), "Groutine Paniced: boom")
}
