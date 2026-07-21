package rdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/logger"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type _GormLogKey struct{}

type _GormLoggerAdapter struct{}

func newLogger() gormLogger.Interface {
	return _GormLoggerAdapter{}
}

func contextWithLogger(ctx context.Context, requestLogger *logger.Logger) context.Context {
	return context.WithValue(ctx, _GormLogKey{}, requestLogger)
}

func (_GormLoggerAdapter) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return _GormLoggerAdapter{}
}

func (_GormLoggerAdapter) Info(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Debug(formatMessage(msg, data...))
}

func (_GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Warn(formatMessage(msg, data...))
}

func (_GormLoggerAdapter) Error(ctx context.Context, msg string, data ...any) {
	getLogger(ctx).Error(formatMessage(msg, data...))
}

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
	return logger.NewLogger(logger.GlobalOption())
}
