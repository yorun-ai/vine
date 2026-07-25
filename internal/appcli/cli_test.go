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
	clearLevelsForTest()

	t.Cleanup(func() {
		os.Args = prevArgs
		argsStdout = prevStdout
		argsStderr = prevStderr
		argsExit = prevExit
		logger.SetGlobalLevel(prevLogLevel)
		clearLevelsForTest()
	})
}

func clearLevelsForTest() {
	for pattern := range logger.Levels() {
		logger.ClearLevel(pattern)
	}
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

func TestHandleSetsNamedRulesWithExactPriority(t *testing.T) {
	resetArgsForTest(t)
	logger.SetGlobalLevel(logger.LevelError)
	os.Args = []string{
		"/tmp/vine",
		"--log-rule", "app:*:rpc:server=WARN",
		"--log-rule", "app:demo.user=INFO",
		"--log-rule", "app:demo.user:rpc:server=DEBUG",
	}
	argsExit = func(int) {}

	Handle()

	if !logger.New("app", "demo.user", "rpc", "server").Enabled(logger.LevelDebug) {
		t.Fatal("expected exact Rpc server DEBUG rule")
	}
	if logger.New("app", "demo.order", "rpc", "server").Enabled(logger.LevelInfo) {
		t.Fatal("wildcard Rpc server WARN rule should reject INFO")
	}
	if !logger.New("app", "demo.user", "task").Enabled(logger.LevelInfo) {
		t.Fatal("App prefix INFO rule should apply to other categories")
	}
}

func TestHandleParsesRepeatedRules(t *testing.T) {
	resetArgsForTest(t)
	os.Args = []string{
		"/tmp/vine",
		"--log-rule", "app:demo.user=WARN",
		"--log-rule", "app:demo.user=DEBUG",
	}
	argsExit = func(int) {}

	Handle()

	if !logger.New("app", "demo.user").Enabled(logger.LevelDebug) {
		t.Fatal("last repeated pattern should win")
	}
}

func TestHandleParsesEnvironmentRules(t *testing.T) {
	resetArgsForTest(t)
	os.Args = []string{"/tmp/vine"}
	t.Setenv(envLogRules, "app:demo.order:event=DEBUG,app:demo.order:task=ERROR")
	argsExit = func(int) {}

	Handle()

	if !logger.New("app", "demo.order", "event").Enabled(logger.LevelDebug) {
		t.Fatal("expected Event override from environment")
	}
	if logger.New("app", "demo.order", "task").Enabled(logger.LevelWarn) {
		t.Fatal("expected Task ERROR override from environment")
	}
}

func TestInvalidRuleDoesNotPartiallyUpdateLevels(t *testing.T) {
	resetArgsForTest(t)
	logger.SetGlobalLevel(logger.LevelInfo)
	logger.SetLevel("app:demo.user", logger.LevelDebug)

	_, err := parseArgs([]string{
		"/tmp/vine",
		"--log-level", "ERROR",
		"--log-rule", "demo.*=DEBUG",
	})
	if err == nil {
		t.Fatal("expected invalid wildcard rule error")
	}
	if logger.GlobalOption().Level != logger.LevelInfo {
		t.Fatal("invalid update must preserve the default level")
	}
	if !logger.New("app", "demo.user").Enabled(logger.LevelDebug) {
		t.Fatal("invalid update must preserve existing level rules")
	}
}

func TestParseRuleRejectsPureWildcardPatterns(t *testing.T) {
	for _, pattern := range []string{"*", "**"} {
		if _, _, err := parseRule(pattern + "=DEBUG"); err == nil {
			t.Fatalf("expected pure wildcard pattern %q to be rejected", pattern)
		}
	}
}
