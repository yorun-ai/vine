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

// CopyResponseBody returns transfer failures instead of treating a partial body as EOF.
func CopyResponseBody(w http.ResponseWriter, response *http.Response) (int64, error) {
	if response.Request != nil && response.Request.Method == http.MethodHead || response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusNotModified || response.StatusCode < 200 {
		return 0, nil
	}
	var n int64
	var err error
	if IsEventStreamResponse(response) {
		n, err = copyStreamBody(w, response.Body)
	} else {
		n, err = io.Copy(_ResponseBodyWriter{w}, response.Body)
	}
	if err == nil {
		if length, known := ResponseBodyLength(response); known && n != length {
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

func copyStreamBody(w http.ResponseWriter, body io.ReadCloser) (int64, error) {
	controller := http.NewResponseController(w)
	buffer := make([]byte, 32*1024)
	var transferred int64
	for {
		n, readErr := readStreamBody(body, buffer, DefaultStreamIdleTimeout)
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

func readStreamBody(body io.ReadCloser, buffer []byte, idleTimeout time.Duration) (int, error) {
	resultCh := make(chan _ReadResult, 1)
	go func() {
		n, err := body.Read(buffer)
		resultCh <- _ReadResult{n: n, err: err}
	}()

	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-timer.C:
		_ = body.Close()
		return 0, context.DeadlineExceeded
	}
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
