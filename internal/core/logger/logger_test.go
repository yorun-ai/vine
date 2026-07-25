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

func TestChildAppendsNameAndInheritsParentState(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelError)
	SetLevel("daemon:link:event:client", LevelDebug)

	parent := New("daemon:link:event").With(slog.String("component", "listener"))
	child := parent.Child("client")

	if parent.Enabled(LevelDebug) {
		t.Fatal("parent should continue using the global fallback")
	}
	if !child.Enabled(LevelDebug) {
		t.Fatal("child should resolve levels using its appended name")
	}
	if parent.writer != child.writer {
		t.Fatal("child should reuse the parent output writer")
	}
	if len(child.attrs) != 1 || child.attrs[0].Key != "component" {
		t.Fatal("child should inherit parent attributes")
	}
}

func TestChildAcceptsColonSeparatedAndMultipleNameSegments(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelError)
	SetLevel("daemon:link:event:client", LevelDebug)

	if !New("daemon:link").Child("event:client").Enabled(LevelDebug) {
		t.Fatal("colon-separated child names should be appended")
	}
	if !New("daemon").Child("link", "event", "client").Enabled(LevelDebug) {
		t.Fatal("multiple child names should be appended")
	}
}

func TestChildRejectsInvalidNameSegments(t *testing.T) {
	parent := New("vine:test")
	for _, name := range []string{"", ":client", "client:", "client::rpc", "*", "**", "client*"} {
		t.Run(name, func(t *testing.T) {
			assertPanics(t, func() {
				parent.Child(name)
			})
		})
	}
}

func TestChildKeepsFixedLevel(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("daemon:link:event:client", LevelDebug)

	parent := New("daemon:link:event", WithOption{Mode: ModeText, Level: LevelInfo})
	if parent.Child("client").Enabled(LevelDebug) {
		t.Fatal("child should inherit the parent's fixed level")
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
