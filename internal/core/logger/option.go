package logger

import (
	"log/slog"
	"os"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

// Mode

type Mode string

const (
	ModeJSON Mode = "JSON"
	ModeText Mode = "TEXT"
)

func IsValidMode(mode Mode) bool {
	return mode == ModeJSON || mode == ModeText
}

// Level

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

func IsValidLevel(level Level) bool {
	return level == LevelDebug ||
		level == LevelInfo ||
		level == LevelWarn ||
		level == LevelError
}

func (l Level) ToSLogLevel() slog.Level {
	vpre.Check(IsValidLevel(l), "%+v is not a valid LogLevel", l)
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Option

type Option struct {
	Mode       Mode
	Level      Level
	OutputPath string
}

// Global

var globalOptionMu sync.RWMutex
var globalOption = newGlobalOption()
var globalLevel slog.LevelVar

func init() {
	globalLevel.Set(globalOption.Level.ToSLogLevel())
}

func newGlobalOption() *Option {
	mode := ModeText
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		mode = ModeJSON
	}

	level := LevelInfo

	return &Option{
		Mode:  mode,
		Level: level,
	}
}

func SetGlobalMode(mode Mode) {
	vpre.Check(IsValidMode(mode), "%+v is not a valid LogMode", mode)
	globalOptionMu.Lock()
	defer globalOptionMu.Unlock()
	globalOption.Mode = mode
}

func SetGlobalLevel(level Level) {
	vpre.Check(IsValidLevel(level), "%+v is not a valid LogLevel", level)
	globalOptionMu.Lock()
	globalOption.Level = level
	globalOptionMu.Unlock()
	globalLevel.Set(level.ToSLogLevel())
}

func GlobalOption() *Option {
	globalOptionMu.RLock()
	defer globalOptionMu.RUnlock()
	return &Option{
		Mode:       globalOption.Mode,
		Level:      globalOption.Level,
		OutputPath: "",
	}
}
