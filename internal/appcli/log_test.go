package appcli

import (
	"os"
	"testing"

	"go.yorun.ai/vine/core/logger"
)

func TestHandleSetsLogLevel(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine", "--log-level", "DEBUG"}
	argsExit = func(int) {}

	Handle()

	if !logger.New("vine:test").Enabled(logger.LevelDebug) {
		t.Fatal("expected global DEBUG level")
	}
}

func TestHandleSetsLogLevelFromEnv(t *testing.T) {
	resetArgsForTest(t)

	os.Args = []string{"/tmp/vine"}
	argsExit = func(int) {}
	t.Setenv(envLogLevel, "WARN")

	Handle()

	log := logger.New("vine:test")
	if log.Enabled(logger.LevelDebug) || !log.Enabled(logger.LevelWarn) {
		t.Fatal("expected global WARN level")
	}
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
	log := logger.New("vine:test")
	if log.Enabled(logger.LevelDebug) || !log.Enabled(logger.LevelInfo) {
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
