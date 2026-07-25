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
		SetGlobalMode(prev.Mode)
		SetGlobalLevel(prev.Level)
	})
}

func TestIsValidMode(t *testing.T) {
	for _, mode := range []Mode{ModeJSON, ModeText} {
		if !IsValidMode(mode) {
			t.Fatalf("expected valid mode: %s", mode)
		}
	}

	if IsValidMode(Mode("PLAIN")) {
		t.Fatal("expected invalid mode")
	}
}

func TestLevelToSLogLevel(t *testing.T) {
	cases := []struct {
		level Level
		want  slog.Level
	}{
		{level: LevelDebug, want: slog.LevelDebug},
		{level: LevelInfo, want: slog.LevelInfo},
		{level: LevelWarn, want: slog.LevelWarn},
		{level: LevelError, want: slog.LevelError},
	}

	for _, tc := range cases {
		if got := tc.level.ToSLogLevel(); got != tc.want {
			t.Fatalf("%s.ToSLogLevel() = %v, want %v", tc.level, got, tc.want)
		}
	}
}

func TestIsValidLevel(t *testing.T) {
	for _, level := range []Level{LevelDebug, LevelInfo, LevelWarn, LevelError} {
		if !IsValidLevel(level) {
			t.Fatalf("expected valid level: %s", level)
		}
	}

	if IsValidLevel(Level("TRACE")) {
		t.Fatal("expected invalid level")
	}
	if IsValidLevel(LevelAuto) {
		t.Fatal("AUTO is an option policy, not a concrete logging threshold")
	}
}

func TestSetGlobalMode(t *testing.T) {
	resetGlobalOptionForTest(t)

	SetGlobalMode(ModeJSON)
	if got := GlobalOption().Mode; got != ModeJSON {
		t.Fatalf("GlobalOption().Mode = %s, want %s", got, ModeJSON)
	}

	SetGlobalMode(ModeText)
	if got := GlobalOption().Mode; got != ModeText {
		t.Fatalf("GlobalOption().Mode = %s, want %s", got, ModeText)
	}
}

func TestSetGlobalModeRejectsInvalidMode(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalMode(Mode("PLAIN"))
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
			SetGlobalMode(ModeJSON)
			SetGlobalLevel(LevelDebug)
			_ = GlobalOption()
			SetGlobalMode(ModeText)
			SetGlobalLevel(LevelInfo)
		}()
	}
	wg.Wait()
}

func TestAutoLoggerAndChildFollowDefaultLevelChanges(t *testing.T) {
	resetGlobalOptionForTest(t)
	SetGlobalLevel(LevelInfo)

	log := New("vine:test", WithOption{Mode: ModeText, Level: LevelAuto})
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
	log := New("vine:test", WithOption{Mode: ModeText, Level: LevelInfo})

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

	SetDefault(New("vine:test", WithOption{Mode: ModeText, Level: LevelAuto}))
	SetGlobalLevel(LevelDebug)
	if !defaultLogger.Enabled(LevelDebug) {
		t.Fatal("default auto logger should follow the default level")
	}

	fixed := New("vine:test", WithOption{Mode: ModeText, Level: LevelInfo})
	SetDefault(fixed)
	if defaultLogger.Enabled(LevelDebug) {
		t.Fatal("explicit fixed default logger should keep its own threshold")
	}
}
