package logger

import (
	"log/slog"
	"testing"
)

func TestWithReturnsChildLogger(t *testing.T) {
	logger := New(GlobalOption())
	child := logger.With(slog.String("group", "child"))

	if logger == child {
		t.Fatal("With should return a child logger")
	}
}

func TestNewJoinsNameSegmentsAndAcceptsFinalWithOption(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("**", LevelError)
	SetLevel("demo.user:rpc:server", LevelDebug)

	auto := New("demo.user:rpc", "server")
	if !auto.Enabled(LevelDebug) {
		t.Fatal("colon-separated arguments should form one matching logger name")
	}

	fixed := New("demo.user", "rpc", "server", WithOption{
		Mode:  ModeText,
		Level: LevelInfo,
	})
	if fixed.Enabled(LevelDebug) {
		t.Fatal("final WithOption should override the dynamic level policy")
	}
}

func TestNewAcceptsColonSeparatedAndWildcardNameSegments(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("**", LevelError)
	SetLevel("a:**:rpc:server", LevelDebug)

	log := New("a:**", "rpc:server")
	if !log.Enabled(LevelDebug) {
		t.Fatal("colon-separated wildcard name segments should be accepted")
	}
}

func TestNewRejectsWithOptionBeforeFinalArgument(t *testing.T) {
	assertPanics(t, func() {
		New("demo.user", WithOption{Mode: ModeText, Level: LevelInfo}, "rpc")
	})
}

func TestNewRejectsInvalidNameSegments(t *testing.T) {
	for _, segment := range []string{"", ":rpc", "rpc:", "rpc::server", "***", "rpc*", "*rpc", "*ab*"} {
		t.Run(segment, func(t *testing.T) {
			assertPanics(t, func() {
				New(segment)
			})
		})
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
