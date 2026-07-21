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
}

func TestSetGlobalLevel(t *testing.T) {
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

func TestSetGlobalLevelRejectsInvalidLevel(t *testing.T) {
	resetGlobalOptionForTest(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	SetGlobalLevel(Level("TRACE"))
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
