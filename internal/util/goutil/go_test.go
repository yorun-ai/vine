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

		GoSafely(func(a, b int) {
			done <- a + b
		}, 2, 5)
		synctest.Wait()

		assert.Equal(t, 7, <-done)
	})
}

func TestGoSafelyRecoversPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		done := make(chan struct{}, 1)

		assert.NotPanics(t, func() {
			GoSafely(func() {
				defer func() {
					done <- struct{}{}
				}()
				panic("boom")
			})
		})
		synctest.Wait()

		<-done
	})
}

func TestRescueWithoutPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		func() {
			defer Rescue()
		}()
	})
}

func TestRescueRecoversPanic(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestRescueHelperProcess$")
	command.Env = append(os.Environ(), "GO_WANT_GOUTIL_RESCUE_HELPER=1")
	output, err := command.CombinedOutput()

	assert.NoError(t, err)
	assert.Contains(t, string(output), "Groutine Paniced: boom")
	assert.NotContains(t, string(output), "FAIL")
}

func TestRescueHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_GOUTIL_RESCUE_HELPER") != "1" {
		return
	}

	assert.NotPanics(t, func() {
		func() {
			defer Rescue()
			panic("boom")
		}()
	})
}
