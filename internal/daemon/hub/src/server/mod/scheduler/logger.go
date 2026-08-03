package scheduler

import "go.yorun.ai/vine/internal/core/logger"

var schedulerLogger = logger.New("daemon:hub:scheduler")

type _SchedulerCronLogger struct{}

func (_SchedulerCronLogger) Info(message string, keysAndValues ...any) {
	schedulerLogger.Debug("scheduler cron "+message, keysAndValues...)
}

func (_SchedulerCronLogger) Error(err error, message string, keysAndValues ...any) {
	args := make([]any, 0, len(keysAndValues)+2)
	args = append(args, keysAndValues...)
	args = append(args, "error", err)
	schedulerLogger.Error("scheduler cron "+message, args...)
}
