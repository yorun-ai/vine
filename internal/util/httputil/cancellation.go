package httputil

import (
	"context"
	"errors"
	"io"

	"go.yorun.ai/vine/internal/core/logger"
)

// IsCanceledResponseTransfer recognizes cancellation propagated through a proxy
// body. EOF alone never proves cancellation; the incoming request must also be
// explicitly canceled. HTTP/2 stream closure additionally requires a downstream
// write/flush marker. Deadlines and unrelated transfer failures remain errors.
func IsCanceledResponseTransfer(ctx context.Context, err error) bool {
	if !errors.Is(ctx.Err(), context.Canceled) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var writeErr *_ResponseWriteError
	if errors.As(err, &writeErr) {
		// net/http and x/net/http2 do not export their stream-closed sentinel.
		// Match only the exact cause of a downstream write/flush failure.
		for cause := writeErr.error; cause != nil; cause = errors.Unwrap(cause) {
			if cause.Error() == "http2: stream closed" {
				return true
			}
		}
	}
	return false
}

// LogResponseTransferError records canceled transfers at Debug and other failures
// at Error through the caller's logger, retaining its transfer diagnostics.
func LogResponseTransferError(ctx context.Context, log *logger.Logger, err error, attrs ...any) {
	logProxyFailure(ctx, log, "web response transfer", err, attrs...)
}

func logProxyFailure(ctx context.Context, log *logger.Logger, operation string, err error, attrs ...any) {
	attrs = append([]any{"error", err}, attrs...)
	if IsCanceledResponseTransfer(ctx, err) {
		log.Debug(operation+" canceled", attrs...)
		return
	}
	log.Error(operation+" failed", attrs...)
}

// LogResponseWriteError records a failure known to originate from the downstream
// response writer. Read and transport failures must use LogResponseTransferError.
func LogResponseWriteError(ctx context.Context, log *logger.Logger, err error, attrs ...any) {
	LogResponseTransferError(ctx, log, &_ResponseWriteError{err}, attrs...)
}

// Retain the direction through body-copy helpers without changing the cause.
type _ResponseWriteError struct{ error }

func (e *_ResponseWriteError) Error() string { return e.error.Error() }

func (e *_ResponseWriteError) Unwrap() error { return e.error }
