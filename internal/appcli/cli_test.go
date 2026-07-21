package appcli

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/internal/core/runtime"
)

func resetArgsForTest(t *testing.T) {
	t.Helper()

	prevArgs := os.Args
	prevStdout := argsStdout
	prevStderr := argsStderr
	prevExit := argsExit
	prevLogLevel := logger.GlobalOption().Level

	t.Cleanup(func() {
		os.Args = prevArgs
		argsStdout = prevStdout
		argsStderr = prevStderr
		argsExit = prevExit
		logger.SetGlobalLevel(prevLogLevel)
	})
}

func testFlag(destination *string) ucli.Flag {
	return &ucli.StringFlag{
		Name:        "endpoint",
		Sources:     ucli.EnvVars("VINE_TEST_ENDPOINT"),
		Destination: destination,
	}
}

func TestHandleIgnoresNonCliArg(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "serve"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(int) {}

	assert.NotPanics(t, func() { Handle() })
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestHandleIgnoresUnknownFlags(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "-test.paniconexit0", "-test.v=true"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(int) {}

	assert.NotPanics(t, func() { Handle() })
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestHandlePrintsVersionAndExits(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "version"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(code int) { panic(fmt.Sprintf("exit:%d", code)) }

	assert.PanicsWithValue(t, "exit:0", func() { Handle() })
	assert.Equal(t, runtime.Inspect(), stdout.String())
	assert.Empty(t, stderr.String())
}

func TestHandlePrintsHelpAndExits(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "help"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(code int) { panic(fmt.Sprintf("exit:%d", code)) }

	assert.PanicsWithValue(t, "exit:0", func() { Handle(testFlag(new(string))) })
	assert.Contains(t, stdout.String(), "application runtime options")
	assert.Contains(t, stdout.String(), "--log-level")
	assert.Contains(t, stdout.String(), "--endpoint")
	assert.Empty(t, stderr.String())
}

func TestHandleIgnoresHelpFlag(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "--help"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(int) {}

	assert.NotPanics(t, func() { Handle(testFlag(new(string))) })
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestHandleParsesFlag(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine", "--endpoint", "http://10.0.0.8:7079"}
	argsExit = func(int) {}

	var endpoint string
	Handle(testFlag(&endpoint))

	assert.Equal(t, "http://10.0.0.8:7079", endpoint)
}

func TestHandleParsesFlagFromEnv(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine"}
	argsExit = func(int) {}
	t.Setenv("VINE_TEST_ENDPOINT", "http://10.0.0.9:7079")

	var endpoint string
	Handle(testFlag(&endpoint))

	assert.Equal(t, "http://10.0.0.9:7079", endpoint)
}

func TestHandleSetsLogLevel(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine", "--log-level", "DEBUG"}
	argsExit = func(int) {}

	Handle()

	assert.Equal(t, logger.LevelDebug, logger.GlobalOption().Level)
}

func TestHandleSetsLogLevelFromEnv(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine"}
	argsExit = func(int) {}
	t.Setenv(envLogLevel, "WARN")

	Handle()

	assert.Equal(t, logger.LevelWarn, logger.GlobalOption().Level)
}
