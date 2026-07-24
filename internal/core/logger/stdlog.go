package logger

import (
	"context"
	"io"
	stdLog "log"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"go.yorun.ai/vine/util/vslice"
)

var standardLoggerWriter io.Writer = &_StandardLoggerWriter{}
var standardLogger *slog.Logger
var stdLogProcessors = vslice.NewMutexSlice[_StdLogProcessorEntry]()

func init() {
	stdLog.SetFlags(0)
	stdLog.SetPrefix("")
	stdLog.SetOutput(standardLoggerWriter)
}

func setStandardLogger(config Option, leveler slog.Leveler) {
	standardLogger = newSlogLogger(&config, true, leveler)
}

type _StandardLoggerWriter struct{}

type _StdLogProcessorEntry struct {
	category  string
	processor StdLogProcessor
}

// StdLogProcessor handles one standard-library log message at a time.
// It returns the processed message, whether the message should be shown,
// and whether the processor matched this message at all.
//
// When matched is false, processedMsg and show are ignored and the next
// processor continues matching the original message.
//
// When matched is true, processor dispatch stops immediately:
//   - show == false means drop this message
//   - show == true means log processedMsg
type StdLogProcessor func(msg string) (processedMsg string, show bool, matched bool)

// StdLogPrefixIdentityProcessor matches messages with the given prefix and
// keeps the original message unchanged.
func StdLogPrefixIdentityProcessor(prefix string) StdLogProcessor {
	return func(msg string) (string, bool, bool) {
		if !strings.HasPrefix(msg, prefix) {
			return "", false, false
		}
		return msg, true, true
	}
}

// StdLogPrefixFilterProcessor matches messages with the given prefix and
// suppresses them.
func StdLogPrefixFilterProcessor(prefix string) StdLogProcessor {
	return func(msg string) (string, bool, bool) {
		if !strings.HasPrefix(msg, prefix) {
			return "", false, false
		}
		return "", false, true
	}
}

// StdLogRegexpIdentityProcessor matches messages by regexp and keeps the
// original message unchanged.
func StdLogRegexpIdentityProcessor(expr string) StdLogProcessor {
	re := regexp.MustCompile(expr)
	return func(msg string) (string, bool, bool) {
		if !re.MatchString(msg) {
			return "", false, false
		}
		return msg, true, true
	}
}

// StdLogRegexpFilterProcessor matches messages by regexp and suppresses them.
func StdLogRegexpFilterProcessor(expr string) StdLogProcessor {
	re := regexp.MustCompile(expr)
	return func(msg string) (string, bool, bool) {
		if !re.MatchString(msg) {
			return "", false, false
		}
		return "", false, true
	}
}

// ConfigureStdLogProcessors configures one or more startup-time processors under the given category.
// Processors are evaluated in registration order across categories, and
// the first matched processor determines the final result.
func ConfigureStdLogProcessors(category string, processors ...StdLogProcessor) {
	for _, processor := range processors {
		stdLogProcessors.Append(_StdLogProcessorEntry{
			category:  category,
			processor: processor,
		})
	}
}

func (*_StandardLoggerWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}
	logStandard(LevelDebug, msg, "")
	processedMsg, show, category := processStdLogMessage(msg)
	if !show {
		return len(p), nil
	}

	logStandard(LevelInfo, processedMsg, category)
	return len(p), nil
}

func processStdLogMessage(msg string) (string, bool, string) {
	for _, entry := range stdLogProcessors.Snapshot() {
		// Unmatched processors are skipped entirely; they cannot rewrite
		// or suppress the message.
		processedMsg, show, matched := entry.processor(msg)
		if !matched {
			continue
		}
		// The first matched processor wins.
		if !show {
			return "", false, entry.category
		}
		return processedMsg, true, entry.category
	}
	return msg, true, ""
}

func logStandard(level Level, msg string, category string) {
	if standardLogger == nil {
		return
	}

	handler := standardLogger.Handler()
	slogLevel := level.ToSLogLevel()
	if !handler.Enabled(context.Background(), slogLevel) {
		return
	}

	record := slog.NewRecord(time.Now(), slogLevel, msg, 0)
	record.AddAttrs(slog.Any(slog.SourceKey, &slog.Source{
		File: sourceOfStandardLog(category),
		Line: 0,
	}))
	_ = handler.Handle(context.Background(), record)
}

func sourceOfStandardLog(category string) string {
	if category == "" {
		return "STDLOG"
	}
	return "STDLOG/" + category
}
