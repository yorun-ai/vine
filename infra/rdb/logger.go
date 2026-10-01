package rdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.yorun.ai/vine/core/logger"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type _GormLogKey struct{}

type _GormLoggerAdapter struct{}

var fallbackLogger = logger.New("vine:infra:rdb")

func newLogger() gormLogger.Interface {
	return _GormLoggerAdapter{}
}

func contextWithLogger(ctx context.Context, requestLogger *logger.Logger) context.Context {
	return context.WithValue(ctx, _GormLogKey{}, requestLogger)
}

// LogMode returns the adapter; logging levels are controlled by the Vine logger
// rather than the requested GORM level.
func (_GormLoggerAdapter) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return _GormLoggerAdapter{}
}

// Info writes a formatted GORM informational message at Vine debug level.
func (_GormLoggerAdapter) Info(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Debug(formatMessage(msg, data...))
}

// Warn writes a formatted GORM warning using the context logger.
func (_GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Warn(formatMessage(msg, data...))
}

// Error writes a formatted GORM error using the context logger.
func (_GormLoggerAdapter) Error(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Error(formatMessage(msg, data...))
}

// Trace logs SQL, affected rows, elapsed time, and any error at Vine debug level.
func (_GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, sqlBuilder func() (string, int64), err error) {
	requestLogger := getLogger(ctx)
	if err != nil {
		requestLogger.Debug(formatTrace(begin, sqlBuilder, err))
		return
	}
	requestLogger.Debug(formatTrace(begin, sqlBuilder, nil))
}

func formatMessage(msg string, data ...any) string {
	return fmt.Sprintf(strings.TrimSuffix(msg, "\n"), data...)
}

func formatTrace(begin time.Time, sqlBuilder func() (string, int64), err error) string {
	sql, rows := sqlBuilder()
	location := shortFileWithLineNum(utils.FileWithLineNum())
	elapsedMillis := float64(time.Since(begin).Nanoseconds()) / 1e6
	if err != nil {
		return fmt.Sprintf("<%s, %d rows, %.3fms> err=[%s], sql=[%s]",
			location, rows, elapsedMillis, err, sql)
	}
	return fmt.Sprintf("<%s, %d rows, %.3fms> sql=(%s)",
		location, rows, elapsedMillis, sql)
}

func shortFileWithLineNum(path string) string {
	comps := strings.Split(path, "/")
	if len(comps) < 2 {
		return path
	}
	return strings.Join(comps[len(comps)-2:], "/")
}

func getLogger(ctx context.Context) *logger.Logger {
	if ctx != nil {
		if requestLogger, ok := ctx.Value(_GormLogKey{}).(*logger.Logger); ok && requestLogger != nil {
			return requestLogger
		}
	}
	return fallbackLogger
}
