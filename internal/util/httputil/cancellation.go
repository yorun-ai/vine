package httputil

import (
	"context"
	"errors"
	"io"

	"go.yorun.ai/vine/internal/core/logger"
)

// IsCanceledResponseTransfer recognizes cancellation propagated through a proxy
// body. EOF alone never proves cancellation; the incoming request must also be
// explicitly canceled. Deadlines and unrelated transfer failures remain errors.
func IsCanceledResponseTransfer(ctx context.Context, err error) bool {
	return errors.Is(ctx.Err(), context.Canceled) &&
		(errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF))
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
