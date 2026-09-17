package appcli

import (
	"os"
	"testing"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/core/logger"
)

func resetArgsForTest(t *testing.T) {
	t.Helper()

	prevArgs := os.Args
	prevStdout := argsStdout
	prevStderr := argsStderr
	prevExit := argsExit
	logger.SetGlobalLevel(logger.LevelInfo)
	clearLevelsForTest()

	t.Cleanup(func() {
		os.Args = prevArgs
		argsStdout = prevStdout
		argsStderr = prevStderr
		argsExit = prevExit
		logger.SetGlobalLevel(logger.LevelInfo)
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
