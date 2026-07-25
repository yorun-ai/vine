package logger

import (
	"strings"
	"testing"
)

func resetRulesForTest(t *testing.T) {
	t.Helper()
	previousRules := rules.Load()
	t.Cleanup(func() {
		rules.Store(previousRules)
	})
	empty, err := newRules(map[string]Level{"**": LevelInfo})
	if err != nil {
		t.Fatal(err)
	}
	rules.Store(empty)
}

func TestNamedLoggerResolvesRulesBySpecificity(t *testing.T) {
	resetRulesForTest(t)
	configured, err := newRules(map[string]Level{
		"**":                     LevelError,
		"*:rpc":                  LevelWarn,
		"*:rpc:server":           LevelError,
		"demo.user":              LevelInfo,
		"demo.user:*":            LevelWarn,
		"demo.user:*:server":     LevelInfo,
		"demo.user:rpc":          LevelWarn,
		"demo.user:rpc:server":   LevelDebug,
		"demo.user:rpc:server:x": LevelError,
	})
	if err != nil {
		t.Fatal(err)
	}
	rules.Store(configured)

	tests := []struct {
		name string
		want Level
	}{
		{name: "demo.user:rpc:server", want: LevelDebug},
		{name: "demo.user:rpc:client", want: LevelWarn},
		{name: "demo.user:event:server", want: LevelInfo},
		{name: "demo.user:event:listener", want: LevelWarn},
		{name: "demo.user", want: LevelInfo},
		{name: "demo.order:rpc:server", want: LevelError},
		{name: "demo.order:rpc:client", want: LevelWarn},
		{name: "demo.order:event", want: LevelError},
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
	return New(args...)
}

func TestNamedLoggerFallsBackAsRulesAreCleared(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("**", LevelError)
	SetLevel("*:event", LevelWarn)
	SetLevel("demo.user", LevelInfo)
	SetLevel("demo.user:event", LevelDebug)

	log := New("demo.user", "event")
	if got := resolvedThreshold(log); got != LevelDebug {
		t.Fatalf("initial threshold = %s, want DEBUG", got)
	}
	ClearLevel("demo.user:event")
	if got := resolvedThreshold(log); got != LevelInfo {
		t.Fatalf("App threshold = %s, want INFO", got)
	}
	ClearLevel("demo.user")
	if got := resolvedThreshold(log); got != LevelWarn {
		t.Fatalf("category threshold = %s, want WARN", got)
	}
	ClearLevel("*:event")
	if got := resolvedThreshold(log); got != LevelError {
		t.Fatalf("global threshold = %s, want ERROR", got)
	}
}

func TestNamedLoggerMatchingIsCaseSensitive(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("**", LevelInfo)
	SetLevel("demo.user", LevelDebug)

	if !New("demo.user").Enabled(LevelDebug) {
		t.Fatal("exact logger name should match")
	}
	if New("Demo.User").Enabled(LevelDebug) {
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
	for _, pattern := range []string{"a:***", "a:****", "a*", "*b", "*ab*", "a:rpc*"} {
		if _, err := newRule(pattern, LevelInfo); err == nil {
			t.Errorf("newRule(%q) returned nil error", pattern)
		}
	}
}

func TestSetLevelRejectsInvalidRuleWithoutChangingLevels(t *testing.T) {
	resetRulesForTest(t)
	SetLevel("**", LevelInfo)
	SetLevel("demo.user", LevelDebug)

	assertPanics(t, func() {
		SetLevel("demo.*", LevelError)
	})

	if !New("demo.user").Enabled(LevelDebug) {
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

func TestDefaultLevelCannotBeCleared(t *testing.T) {
	resetRulesForTest(t)

	assertPanics(t, func() {
		ClearLevel("**")
	})
	if Levels()["**"] != LevelInfo {
		t.Fatal("default level should remain configured")
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
