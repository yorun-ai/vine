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
	// LevelAuto follows matching named level rules and falls back to the
	// process-wide global level.
	LevelAuto  Level = "AUTO"
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

func isValidOptionLevel(level Level) bool {
	return level == LevelAuto || IsValidLevel(level)
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

// WithOption configures the logger created by New.
type WithOption struct {
	Mode       Mode
	Level      Level
	OutputPath string
}

var globalModeMu sync.RWMutex
var globalMode = newGlobalMode()

func newGlobalMode() Mode {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		return ModeJSON
	}
	return ModeText
}

func SetGlobalMode(mode Mode) {
	vpre.Check(IsValidMode(mode), "%+v is not a valid LogMode", mode)
	globalModeMu.Lock()
	defer globalModeMu.Unlock()
	globalMode = mode
}

func GlobalOption() *WithOption {
	globalModeMu.RLock()
	defer globalModeMu.RUnlock()
	return &WithOption{
		Mode:  globalMode,
		Level: levelFromSlog(globalLevel.Level()),
	}
}

func levelFromSlog(level slog.Level) Level {
	switch level {
	case slog.LevelDebug:
		return LevelDebug
	case slog.LevelInfo:
		return LevelInfo
	case slog.LevelWarn:
		return LevelWarn
	case slog.LevelError:
		return LevelError
	default:
		panic("global logger level is invalid")
	}
}

type _LevelerFunc func() slog.Level

func (f _LevelerFunc) Level() slog.Level {
	return f()
}

func newLeveler(level Level, nameSegments []string) slog.Leveler {
	if level != LevelAuto {
		return level.ToSLogLevel()
	}
	return _LevelerFunc(func() slog.Level {
		level, ok := rules.Load().resolve(nameSegments)
		if ok {
			return level.ToSLogLevel()
		}
		return globalLevel.Level()
	})
}
