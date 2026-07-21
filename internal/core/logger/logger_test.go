package logger

import (
	"log/slog"
	"testing"
)

func TestWithReturnsChildLogger(t *testing.T) {
	logger := NewLogger(GlobalOption())
	child := logger.With(slog.String("scope", "child"))

	if logger == child {
		t.Fatal("With should return a child logger")
	}
}
