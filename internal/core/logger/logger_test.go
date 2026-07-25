package logger

import (
	"log/slog"
	"testing"
)

func TestWithReturnsChildLogger(t *testing.T) {
	logger := New("vine:test", GlobalOption())
	child := logger.With(slog.String("group", "child"))

	if logger == child {
		t.Fatal("With should return a child logger")
	}
}

func TestNewJoinsNameSegmentsAndAcceptsFinalWithOption(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelError)
	SetLevel("app:demo.user:rpc:server", LevelDebug)

	auto := New("app:demo.user:rpc", "server")
	if !auto.Enabled(LevelDebug) {
		t.Fatal("colon-separated arguments should form one matching logger name")
	}

	fixed := New("app", "demo.user", "rpc", "server", WithOption{
		Mode:  ModeText,
		Level: LevelInfo,
	})
	if fixed.Enabled(LevelDebug) {
		t.Fatal("final WithOption should override the dynamic level policy")
	}
}

func TestNewRejectsWildcardNameSegments(t *testing.T) {
	resetRulesForTest(t)
	assertPanics(t, func() { New("a:**", "rpc:server") })
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
