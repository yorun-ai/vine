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
	logger.ReplaceLevelOverrides(logger.LevelOverrides{})

	t.Cleanup(func() {
		os.Args = prevArgs
		argsStdout = prevStdout
		argsStderr = prevStderr
		argsExit = prevExit
		logger.SetGlobalLevel(prevLogLevel)
		logger.ReplaceLevelOverrides(logger.LevelOverrides{})
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

func TestHandleSetsScopedLogLevelsWithExactPriority(t *testing.T) {
	resetArgsForTest(t)
	logger.SetGlobalLevel(logger.LevelError)
	os.Args = []string{
		"/tmp/vine",
		"--rpc-server-log-level", "WARN",
		"--app-log-level", "demo.user=INFO",
		"--app-scope-log-level", "demo.user:rpc-server=DEBUG",
	}
	argsExit = func(int) {}

	Handle()

	if !logger.NewScopedLogger(logger.Scope{AppName: "demo.user", Subsystem: logger.SubsystemRpcServer}).Enabled(logger.LevelDebug) {
		t.Fatal("expected App plus Rpc server DEBUG override")
	}
	if logger.NewScopedLogger(logger.Scope{AppName: "demo.order", Subsystem: logger.SubsystemRpcServer}).Enabled(logger.LevelInfo) {
		t.Fatal("subsystem WARN override should reject INFO")
	}
	if !logger.NewScopedLogger(logger.Scope{AppName: "demo.user", Subsystem: logger.SubsystemTask}).Enabled(logger.LevelInfo) {
		t.Fatal("App INFO override should apply to other subsystems")
	}
}

func TestHandleParsesRepeatedAndEnvironmentScopedRules(t *testing.T) {
	resetArgsForTest(t)
	os.Args = []string{
		"/tmp/vine",
		"--app-log-level", "demo.user=WARN",
		"--app-log-level", "demo.user=DEBUG",
	}
	t.Setenv(envAppScopeLogLevels, "demo.order:event=DEBUG,demo.order:task=ERROR")
	argsExit = func(int) {}

	Handle()

	if !logger.NewScopedLogger(logger.Scope{AppName: "demo.user"}).Enabled(logger.LevelDebug) {
		t.Fatal("last repeated App selector should win")
	}
	if !logger.NewScopedLogger(logger.Scope{AppName: "demo.order", Subsystem: logger.SubsystemEvent}).Enabled(logger.LevelDebug) {
		t.Fatal("expected Event override from environment")
	}
	if logger.NewScopedLogger(logger.Scope{AppName: "demo.order", Subsystem: logger.SubsystemTask}).Enabled(logger.LevelWarn) {
		t.Fatal("expected Task ERROR override from environment")
	}
}

func TestInvalidScopedRuleDoesNotPartiallyUpdateLevels(t *testing.T) {
	resetArgsForTest(t)
	logger.SetGlobalLevel(logger.LevelInfo)
	logger.SetAppLevel("demo.user", logger.LevelDebug)

	_, err := parseArgs([]string{
		"/tmp/vine",
		"--log-level", "ERROR",
		"--app-scope-log-level", "demo.user:unknown=DEBUG",
	})
	if err == nil {
		t.Fatal("expected invalid scoped rule error")
	}
	if logger.GlobalOption().Level != logger.LevelInfo {
		t.Fatal("invalid update must preserve global level")
	}
	if !logger.NewScopedLogger(logger.Scope{AppName: "demo.user"}).Enabled(logger.LevelDebug) {
		t.Fatal("invalid update must preserve existing scoped snapshot")
	}
}
