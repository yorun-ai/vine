package appcli

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/buildinfo"
)

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

func TestHandlePrintsVersionAndExits(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "version"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(code int) { panic(fmt.Sprintf("exit:%d", code)) }

	assert.PanicsWithValue(t, "exit:0", func() { Handle() })
	assert.Equal(t, buildinfo.Inspect(), stdout.String())
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
