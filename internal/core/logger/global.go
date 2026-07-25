package logger

import (
	"log/slog"
	"os"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

var globalFormatMu sync.RWMutex
var globalFormat = newGlobalFormat()

// globalLevel is the fallback for auto-level loggers that match no named rule.
// slog.LevelVar's zero value is INFO.
var globalLevel slog.LevelVar

func newGlobalFormat() Format {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		return FormatJSON
	}
	return FormatText
}

func SetGlobalFormat(format Format) {
	vpre.Check(IsValidFormat(format), "%+v is not a valid log format", format)
	globalFormatMu.Lock()
	defer globalFormatMu.Unlock()
	globalFormat = format
}

func SetGlobalLevel(level Level) {
	vpre.Check(IsValidLevel(level), "%+v is not a valid LogLevel", level)
	globalLevel.Set(level.ToSLogLevel())
}

func GlobalOption() *WithOption {
	globalFormatMu.RLock()
	defer globalFormatMu.RUnlock()
	return &WithOption{
		Format: globalFormat,
		Level:  levelFromSlog(globalLevel.Level()),
	}
}
