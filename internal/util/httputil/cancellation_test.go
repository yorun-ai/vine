package httputil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestIsCanceledResponseTransfer(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	deadline, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"client cancel", canceled, context.Canceled, true},
		{"canceled HTTP2 write", canceled, &_ResponseWriteError{errors.New("http2: stream closed")}, true},
		{"wrapped HTTP2 write", canceled, fmt.Errorf("copy: %w", &_ResponseWriteError{fmt.Errorf("write: %w", errors.New("http2: stream closed"))}), true},
		{"live HTTP2 write", context.Background(), &_ResponseWriteError{errors.New("http2: stream closed")}, false},
		{"deadline HTTP2 write", deadline, &_ResponseWriteError{errors.New("http2: stream closed")}, false},
		{"upstream HTTP2 closed", canceled, errors.New("http2: stream closed"), false},
		{"HTTP2 internal error", canceled, &_ResponseWriteError{errors.New("stream error: stream ID 189; INTERNAL_ERROR; received from peer")}, false},
		{"similar error", canceled, &_ResponseWriteError{errors.New("http2: stream closed unexpectedly")}, false},
		{"canceled proxy EOF", canceled, io.EOF, true},
		{"canceled proxy truncated stream", canceled, fmt.Errorf("body: %w", io.ErrUnexpectedEOF), true},
		{"live EOF", context.Background(), io.EOF, false},
		{"live truncated stream", context.Background(), io.ErrUnexpectedEOF, false},
		{"upstream cancellation", context.Background(), context.Canceled, false},
		{"deadline EOF", deadline, io.ErrUnexpectedEOF, false},
		{"deadline", deadline, context.DeadlineExceeded, false},
		{"unrelated failure after cancellation", canceled, errors.New("disk failure"), false},
		{"short write after cancellation", canceled, io.ErrShortWrite, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCanceledResponseTransfer(tc.ctx, tc.err); got != tc.want {
				t.Fatalf("IsCanceledResponseTransfer = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLogResponseTransferError(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			log, logPath := newTestProxyLogger(t, "daemon:portal:webgw")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			level := "ERROR"
			if canceled {
				cancel()
				level = "DEBUG"
			}
			LogResponseTransferError(ctx, log, io.ErrUnexpectedEOF, "phase", "body", "expectedLength", 10, "transferredBytes", 7)
			record := readProxyLog(t, logPath)
			if record["logger"] != log.Name() || record["level"] != level || record["phase"] != "body" || record["error"] != io.ErrUnexpectedEOF.Error() || record["expectedLength"] != float64(10) || record["transferredBytes"] != float64(7) {
				t.Fatalf("unexpected transfer log: %v", record)
			}
		})
	}
}

func TestLogResponseWriteError(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			log, path := newTestProxyLogger(t, "daemon:portal:webgw")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			level := "ERROR"
			if canceled {
				cancel()
				level = "DEBUG"
			}
			LogResponseWriteError(ctx, log, errors.New("http2: stream closed"), "phase", "write", "transferredBytes", 0)
			record := readProxyLog(t, path)
			if record["level"] != level || record["error"] != "http2: stream closed" || record["phase"] != "write" {
				t.Fatalf("unexpected write log: %v", record)
			}
		})
	}
}
