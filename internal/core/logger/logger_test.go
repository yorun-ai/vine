package logger

import (
	"log/slog"
	"sync"
	"testing"
)

func resetGlobalOptionForTest(t *testing.T) {
	t.Helper()

	previousFormat := currentGlobalFormat()
	previousLevel := globalLevel.Level()
	previousOutputPath := globalWriter.OutputPath()
	t.Cleanup(func() {
		SetGlobalFormat(previousFormat)
		globalLevel.Set(previousLevel)
		SetGlobalOutputPath(previousOutputPath)
	})
}

func TestGlobalSettingsConcurrentAccess(t *testing.T) {
	resetGlobalOptionForTest(t)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			SetGlobalFormat(FormatJSON)
			SetGlobalLevel(LevelDebug)
			_ = currentGlobalFormat()
			_ = globalLevel.Level()
			_ = globalWriter.OutputPath()
			SetGlobalFormat(FormatText)
			SetGlobalLevel(LevelInfo)
		})
	}
	wg.Wait()
}

func TestWithReturnsChildLogger(t *testing.T) {
	logger := New("vine:test")
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
	if parent.Name() != "daemon:link:event" {
		t.Fatalf("parent.Name() = %q", parent.Name())
	}
	if child.Name() != "daemon:link:event:client" {
		t.Fatalf("child.Name() = %q", child.Name())
	}

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

func TestLoggerRejectsReservedLoggerAttr(t *testing.T) {
	log := New("vine:test", WithOption{Format: FormatText, Level: LevelInfo})

	assertPanics(t, func() {
		log.With(slog.String(loggerKey, "custom"))
	})
	assertPanics(t, func() {
		log.Info("reserved string key", loggerKey, "custom")
	})
	assertPanics(t, func() {
		log.Info("reserved attr", slog.String(loggerKey, "custom"))
	})
	assertPanics(t, func() {
		log.With(slog.Group("", slog.String(loggerKey, "custom")))
	})
}

func TestLoggerAllowsLoggerAttrInNamedGroup(t *testing.T) {
	log := New("vine:test").With(
		slog.Group("nested", slog.String(loggerKey, "custom")),
	)
	if log.Name() != "vine:test" {
		t.Fatalf("log.Name() = %q", log.Name())
	}
}

func TestChildKeepsFixedLevel(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("daemon:link:event:client", LevelDebug)

	parent := New("daemon:link:event", WithOption{Format: FormatText, Level: LevelInfo})
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
		Format: FormatText,
		Level:  LevelInfo,
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
		New("demo.user", WithOption{Format: FormatText, Level: LevelInfo}, "rpc")
	})
}

func TestNewRejectsWithOptionPointer(t *testing.T) {
	assertPanics(t, func() {
		New("demo.user", &WithOption{})
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
