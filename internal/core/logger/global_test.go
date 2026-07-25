package logger

import (
	"log/slog"
	"sync"
	"testing"
)

func resetGlobalOptionForTest(t *testing.T) {
	t.Helper()

	prev := GlobalOption()
	t.Cleanup(func() {
		SetGlobalFormat(prev.Format)
		SetGlobalLevel(prev.Level)
	})
}

func TestSetGlobalFormat(t *testing.T) {
	resetGlobalOptionForTest(t)

	SetGlobalFormat(FormatJSON)
	if got := GlobalOption().Format; got != FormatJSON {
		t.Fatalf("GlobalOption().Format = %s, want %s", got, FormatJSON)
	}

	SetGlobalFormat(FormatText)
	if got := GlobalOption().Format; got != FormatText {
		t.Fatalf("GlobalOption().Format = %s, want %s", got, FormatText)
	}
}

func TestSetGlobalFormatRejectsInvalidFormat(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalFormat(Format("PLAIN"))
}

func TestGlobalOptionReadsDefaultLevel(t *testing.T) {
	resetGlobalOptionForTest(t)

	SetGlobalLevel(LevelDebug)
	if got := GlobalOption().Level; got != LevelDebug {
		t.Fatalf("GlobalOption().Level = %s, want %s", got, LevelDebug)
	}

	SetGlobalLevel(LevelError)
	if got := GlobalOption().Level; got != LevelError {
		t.Fatalf("GlobalOption().Level = %s, want %s", got, LevelError)
	}
}

func TestDefaultLevelRejectsAuto(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalLevel(LevelAuto)
}

func TestGlobalOptionConcurrentAccess(t *testing.T) {
	resetGlobalOptionForTest(t)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SetGlobalFormat(FormatJSON)
			SetGlobalLevel(LevelDebug)
			_ = GlobalOption()
			SetGlobalFormat(FormatText)
			SetGlobalLevel(LevelInfo)
		}()
	}
	wg.Wait()
}

func TestAutoLoggerAndChildFollowDefaultLevelChanges(t *testing.T) {
	resetGlobalOptionForTest(t)
	SetGlobalLevel(LevelInfo)

	log := New("vine:test", WithOption{Format: FormatText, Level: LevelAuto})
	child := log.With(slog.String("group", "child"))
	if log.Enabled(LevelDebug) || child.Enabled(LevelDebug) {
		t.Fatal("Debug should initially be disabled")
	}

	SetGlobalLevel(LevelDebug)
	if !log.Enabled(LevelDebug) || !child.Enabled(LevelDebug) {
		t.Fatal("existing auto loggers should follow the default level")
	}
}

func TestFixedLoggerDoesNotFollowLevelChanges(t *testing.T) {
	resetGlobalOptionForTest(t)
	SetGlobalLevel(LevelInfo)
	log := New("vine:test", WithOption{Format: FormatText, Level: LevelInfo})

	SetGlobalLevel(LevelDebug)
	if log.Enabled(LevelDebug) {
		t.Fatal("fixed logger should keep its configured level")
	}
}

func TestDefaultLoggerKeepsInjectedLoggerLevelSemantics(t *testing.T) {
	resetGlobalOptionForTest(t)
	previousDefault := defaultLogger
	t.Cleanup(func() { SetDefault(previousDefault) })
	SetGlobalLevel(LevelInfo)

	SetDefault(New("vine:test", WithOption{Format: FormatText, Level: LevelAuto}))
	SetGlobalLevel(LevelDebug)
	if !defaultLogger.Enabled(LevelDebug) {
		t.Fatal("default auto logger should follow the default level")
	}

	fixed := New("vine:test", WithOption{Format: FormatText, Level: LevelInfo})
	SetDefault(fixed)
	if defaultLogger.Enabled(LevelDebug) {
		t.Fatal("explicit fixed default logger should keep its own threshold")
	}
}
