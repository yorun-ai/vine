package httputil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ErrStreamIdleTimeout indicates an idle SSE response that may end normally.
// Fixed-length responses instead retain a truncation error.
var ErrStreamIdleTimeout = errors.New("SSE stream idle timeout")

// CopyResponseBody returns transfer failures instead of treating a partial body as EOF.
func CopyResponseBody(w http.ResponseWriter, response *http.Response) (int64, error) {
	return CopyResponseBodyWithIdleTimeout(w, response, 0)
}

// CopyResponseBodyWithIdleTimeout applies an SSE idle limit at the external gateway.
// A zero duration leaves internal forwarding unbounded by idle time.
func CopyResponseBodyWithIdleTimeout(w http.ResponseWriter, response *http.Response, idleTimeout time.Duration) (int64, error) {
	if response.Request != nil && response.Request.Method == http.MethodHead || response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusNotModified || response.StatusCode < 200 {
		return 0, nil
	}
	var n int64
	var err error
	if IsEventStreamResponse(response) {
		ctx := context.Background()
		if response.Request != nil {
			ctx = response.Request.Context()
		}
		n, err = copyStreamBody(ctx, w, response.Body, idleTimeout)
	} else {
		n, err = io.Copy(_ResponseBodyWriter{w}, response.Body)
	}
	if length, known := ResponseBodyLength(response); known {
		switch {
		case errors.Is(err, ErrStreamIdleTimeout):
			err = fmt.Errorf("response idle before completion: %w", io.ErrUnexpectedEOF)
		case err == nil && n != length:
			err = fmt.Errorf("response length mismatch: %w", io.ErrUnexpectedEOF)
		}
	}
	return n, err
}

// ResponseBodyLength reports the declared length, including in-process responses.
func ResponseBodyLength(response *http.Response) (int64, bool) {
	if value := response.Header.Get("Content-Length"); value != "" {
		length, err := strconv.ParseInt(value, 10, 64)
		return length, err == nil && length >= 0
	}
	return response.ContentLength, response.ContentLength > 0
}

func ReadResponseBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return body, err
	}
	if length, known := ResponseBodyLength(response); known && int64(len(body)) != length {
		return body, io.ErrUnexpectedEOF
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func copyStreamBody(ctx context.Context, w http.ResponseWriter, body io.ReadCloser, idleTimeout time.Duration) (int64, error) {
	controller := http.NewResponseController(w)
	buffer := make([]byte, 32*1024)
	read := body.Read
	if idleTimeout > 0 {
		var cleanup func()
		read, cleanup = newIdleStreamReader(ctx, body, idleTimeout)
		defer cleanup()
	}
	var transferred int64
	for {
		n, readErr := read(buffer)
		if n > 0 {
			written, err := w.Write(buffer[:n])
			transferred += int64(written)
			if err != nil {
				return transferred, &_ResponseWriteError{err}
			}
			if written != n {
				return transferred, io.ErrShortWrite
			}
			if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
				return transferred, &_ResponseWriteError{err}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				readErr = nil
			}
			return transferred, readErr
		}
	}
}

type _ReadResult struct {
	n   int
	err error
}

// Reads are requested only after the previous chunk has been written and flushed,
// so the worker never reuses a buffer still owned by the response writer.
func newIdleStreamReader(ctx context.Context, body io.ReadCloser, idleTimeout time.Duration) (func([]byte) (int, error), func()) {
	requests := make(chan []byte)
	results := make(chan _ReadResult, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for buffer := range requests {
			n, err := body.Read(buffer)
			results <- _ReadResult{n: n, err: err}
		}
	}()
	timer := time.NewTimer(idleTimeout)
	timer.Stop()
	cleanup := func() {
		timer.Stop()
		close(requests)
		// HTTP response bodies must unblock Read on Close. The single buffered
		// result lets an outstanding read finish even after the caller returns.
		_ = body.Close()
		<-finished
	}
	read := func(buffer []byte) (int, error) {
		timer.Reset(idleTimeout)
		defer timer.Stop()
		requests <- buffer
		select {
		case result := <-results:
			return result.n, result.err
		case <-timer.C:
			// Determine the cause before cleanup: Close can itself cancel ctx.
			err := ctx.Err()
			if err == nil {
				if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
					err = context.DeadlineExceeded
				}
			}
			if err == nil {
				select {
				case result := <-results:
					return result.n, result.err
				default:
				}
				err = ErrStreamIdleTimeout
			}
			return 0, err
		}
	}
	return read, cleanup
}

// Avoid io.Copy fast paths bypassing the write-error direction marker.
type _ResponseBodyWriter struct{ io.Writer }

func (w _ResponseBodyWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err != nil {
		return n, &_ResponseWriteError{err}
	}
	return n, nil
}
