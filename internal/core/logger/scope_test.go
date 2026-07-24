package logger

import (
	"log/slog"
	"testing"
)

func resetLevelsForTest(t *testing.T) {
	t.Helper()
	previousGlobal := GlobalOption().Level
	previousSnapshot := levelSnapshot.Load()
	t.Cleanup(func() {
		SetGlobalLevel(previousGlobal)
		levelSnapshot.Store(previousSnapshot)
	})
	ReplaceLevelOverrides(LevelOverrides{})
}

func TestGlobalLoggerAndChildFollowLevelChanges(t *testing.T) {
	resetLevelsForTest(t)
	SetGlobalLevel(LevelInfo)

	log := NewGlobalLogger()
	child := log.With(slog.String("scope", "child"))
	if log.Enabled(LevelDebug) || child.Enabled(LevelDebug) {
		t.Fatal("Debug should initially be disabled")
	}

	SetGlobalLevel(LevelDebug)
	if !log.Enabled(LevelDebug) || !child.Enabled(LevelDebug) {
		t.Fatal("existing global loggers should follow SetGlobalLevel")
	}
}

func TestFixedLoggerDoesNotFollowLevelChanges(t *testing.T) {
	resetLevelsForTest(t)
	SetGlobalLevel(LevelInfo)
	log := NewLogger(&Option{Mode: ModeText, Level: LevelInfo})

	SetGlobalLevel(LevelDebug)
	SetAppLevel("demo.user", LevelDebug)
	if log.Enabled(LevelDebug) {
		t.Fatal("fixed logger should keep its configured level")
	}
}

func TestDefaultLoggerKeepsInjectedLoggerLevelSemantics(t *testing.T) {
	resetLevelsForTest(t)
	previousDefault := defaultLogger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetGlobalLevel(LevelInfo)

	SetDefault(NewGlobalLogger())
	SetGlobalLevel(LevelDebug)
	if !defaultLogger.Enabled(LevelDebug) {
		t.Fatal("default global logger should follow SetGlobalLevel")
	}

	fixed := NewLogger(&Option{Mode: ModeText, Level: LevelInfo})
	SetDefault(fixed)
	if defaultLogger.Enabled(LevelDebug) {
		t.Fatal("explicit fixed default logger should keep its own threshold")
	}
}

func TestScopedLoggerResolvesPriorityAndClearFallback(t *testing.T) {
	resetLevelsForTest(t)
	SetGlobalLevel(LevelError)
	SetSubsystemLevel(SubsystemEvent, LevelWarn)
	SetAppLevel("demo.user", LevelInfo)
	SetAppSubsystemLevel("demo.user", SubsystemEvent, LevelDebug)

	log := NewScopedLogger(Scope{AppName: "demo.user", Subsystem: SubsystemEvent})
	if !log.Enabled(LevelDebug) {
		t.Fatal("App plus subsystem override should have highest dynamic priority")
	}
	ClearAppSubsystemLevel("demo.user", SubsystemEvent)
	if log.Enabled(LevelDebug) || !log.Enabled(LevelInfo) {
		t.Fatal("clearing App plus subsystem should fall back to App override")
	}
	ClearAppLevel("demo.user")
	if log.Enabled(LevelInfo) || !log.Enabled(LevelWarn) {
		t.Fatal("clearing App override should fall back to subsystem override")
	}
	ClearSubsystemLevel(SubsystemEvent)
	if log.Enabled(LevelWarn) || !log.Enabled(LevelError) {
		t.Fatal("clearing subsystem override should fall back to global")
	}
}

func TestScopedLoggerUsesExactCaseSensitiveAppName(t *testing.T) {
	resetLevelsForTest(t)
	SetGlobalLevel(LevelInfo)
	SetAppLevel("demo.user", LevelDebug)

	if !NewScopedLogger(Scope{AppName: "demo.user"}).Enabled(LevelDebug) {
		t.Fatal("exact App name should match")
	}
	if NewScopedLogger(Scope{AppName: "demo"}).Enabled(LevelDebug) {
		t.Fatal("App prefix should not match")
	}
	if NewScopedLogger(Scope{AppName: "Demo.User"}).Enabled(LevelDebug) {
		t.Fatal("App matching should be case-sensitive")
	}
}

func TestReplaceLevelOverridesRejectsInvalidSnapshotAtomically(t *testing.T) {
	resetLevelsForTest(t)
	SetGlobalLevel(LevelInfo)
	SetAppLevel("demo.user", LevelDebug)

	func() {
		defer func() { _ = recover() }()
		ReplaceLevelOverrides(LevelOverrides{
			Apps: map[string]Level{"demo.user": LevelError},
			Subsystems: map[Subsystem]Level{
				Subsystem("unknown"): LevelDebug,
			},
		})
	}()

	if !NewScopedLogger(Scope{AppName: "demo.user"}).Enabled(LevelDebug) {
		t.Fatal("invalid replacement should keep last-known-good overrides")
	}
}
