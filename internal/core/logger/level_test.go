package logger

import (
	"log/slog"
	"strings"
	"testing"
)

func resetRulesForTest(t *testing.T) {
	t.Helper()
	previousRules := rules.Load()
	previousLevel := globalLevel.Level()
	t.Cleanup(func() {
		rules.Store(previousRules)
		globalLevel.Set(previousLevel)
	})
	empty, err := newRules(nil)
	if err != nil {
		t.Fatal(err)
	}
	rules.Store(empty)
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

func TestNamedLoggerResolvesRulesBySpecificity(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelError)
	configured, err := newRules(map[string]Level{
		"app:*:rpc":                  LevelWarn,
		"app:*:rpc:server":           LevelError,
		"app:demo.user":              LevelInfo,
		"app:demo.user:*":            LevelWarn,
		"app:demo.user:*:server":     LevelInfo,
		"app:demo.user:rpc":          LevelWarn,
		"app:demo.user:rpc:server":   LevelDebug,
		"app:demo.user:rpc:server:x": LevelError,
	})
	if err != nil {
		t.Fatal(err)
	}
	rules.Store(configured)

	tests := []struct {
		name string
		want Level
	}{
		{name: "app:demo.user:rpc:server", want: LevelDebug},
		{name: "app:demo.user:rpc:client", want: LevelWarn},
		{name: "app:demo.user:event:server", want: LevelInfo},
		{name: "app:demo.user:event:listener", want: LevelWarn},
		{name: "app:demo.user", want: LevelInfo},
		{name: "app:demo.order:rpc:server", want: LevelError},
		{name: "app:demo.order:rpc:client", want: LevelWarn},
		{name: "app:demo.order:event", want: LevelError},
	}
	for _, test := range tests {
		if got := resolvedThreshold(newNamedLoggerForTest(test.name)); got != test.want {
			t.Errorf("New(%q) threshold = %s, want %s", test.name, got, test.want)
		}
	}
}

func newNamedLoggerForTest(name string) *Logger {
	segments := strings.Split(name, ":")
	args := make([]any, len(segments))
	for index, segment := range segments {
		args[index] = segment
	}
	return New(args[0].(string), args[1:]...)
}

func TestNamedLoggerFallsBackAsRulesAreCleared(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelError)
	SetLevel("app:*:event", LevelWarn)
	SetLevel("app:demo.user", LevelInfo)
	SetLevel("app:demo.user:event", LevelDebug)

	log := New("app", "demo.user", "event")
	if got := resolvedThreshold(log); got != LevelDebug {
		t.Fatalf("initial threshold = %s, want DEBUG", got)
	}
	ClearLevel("app:demo.user:event")
	if got := resolvedThreshold(log); got != LevelInfo {
		t.Fatalf("App threshold = %s, want INFO", got)
	}
	ClearLevel("app:demo.user")
	if got := resolvedThreshold(log); got != LevelWarn {
		t.Fatalf("category threshold = %s, want WARN", got)
	}
	ClearLevel("app:*:event")
	if got := resolvedThreshold(log); got != LevelError {
		t.Fatalf("global threshold = %s, want ERROR", got)
	}
}

func TestNamedLoggerMatchingIsCaseSensitive(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelInfo)
	SetLevel("app:demo.user", LevelDebug)

	if !New("app", "demo.user").Enabled(LevelDebug) {
		t.Fatal("exact logger name should match")
	}
	if New("app", "Demo.User").Enabled(LevelDebug) {
		t.Fatal("logger name matching should be case-sensitive")
	}
}

func TestRuleWildcardsMatchOneOrAnyNumberOfSegments(t *testing.T) {
	oneSegment, err := newRule("a:*:leaf", LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	multipleSegments, err := newRule("a:**:leaf", LevelInfo)
	if err != nil {
		t.Fatal(err)
	}

	if !oneSegment.matches([]string{"a", "one", "leaf"}) {
		t.Fatal("* should match exactly one segment")
	}
	if oneSegment.matches([]string{"a", "one", "two", "leaf"}) {
		t.Fatal("* must not consume multiple segments")
	}
	if !multipleSegments.matches([]string{"a", "one", "two", "leaf"}) {
		t.Fatal("** should match multiple consecutive segments")
	}
	if !multipleSegments.matches([]string{"a", "leaf"}) {
		t.Fatal("** should match zero segments")
	}
}

func TestRulePatternAcceptsOnlyWholeSegmentWildcards(t *testing.T) {
	for _, pattern := range []string{"a:*", "a:**", "**:rpc:server"} {
		if _, err := newRule(pattern, LevelInfo); err != nil {
			t.Errorf("newRule(%q) returned unexpected error: %v", pattern, err)
		}
	}
	for _, pattern := range []string{"*", "**", "a:***", "a:****", "a*", "*b", "*ab*", "a:rpc*"} {
		if _, err := newRule(pattern, LevelInfo); err == nil {
			t.Errorf("newRule(%q) returned nil error", pattern)
		}
	}
}

func TestSetLevelRejectsInvalidRuleWithoutChangingLevels(t *testing.T) {
	resetRulesForTest(t)
	SetGlobalLevel(LevelInfo)
	SetLevel("app:demo.user", LevelDebug)

	assertPanics(t, func() {
		SetLevel("demo.*", LevelError)
	})
	assertPanics(t, func() {
		SetLevel("*", LevelError)
	})
	assertPanics(t, func() {
		SetLevel("**", LevelError)
	})

	if !New("app", "demo.user").Enabled(LevelDebug) {
		t.Fatal("invalid rule should keep existing levels")
	}
}

func TestLevelsReturnsIndependentCopy(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("demo.user", LevelDebug)

	levels := Levels()
	levels["demo.user"] = LevelError
	levels["demo.order"] = LevelWarn

	current := Levels()
	if current["demo.user"] != LevelDebug {
		t.Fatal("mutating returned levels must not change configured levels")
	}
	if _, exists := current["demo.order"]; exists {
		t.Fatal("mutating returned levels must not add configured levels")
	}
}

func TestClearLevelRejectsPureWildcardPatterns(t *testing.T) {
	resetRulesForTest(t)

	for _, pattern := range []string{"*", "**"} {
		assertPanics(t, func() {
			ClearLevel(pattern)
		})
	}
	if len(Levels()) != 0 {
		t.Fatal("invalid clear must not add or remove rules")
	}
}

func resolvedThreshold(log *Logger) Level {
	for _, level := range []Level{LevelDebug, LevelInfo, LevelWarn, LevelError} {
		if log.Enabled(level) {
			return level
		}
	}
	return ""
}
