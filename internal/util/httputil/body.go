package httputil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

func CopyResponseBody(w http.ResponseWriter, response *http.Response) {
	if IsEventStreamResponse(response) {
		copyStreamBody(w, response.Body)
		return
	}
	copyPlainBody(w, response.Body)
}

func copyPlainBody(w http.ResponseWriter, body io.ReadCloser) {
	_, _ = io.Copy(w, body)
}

func ReadResponseBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func copyStreamBody(w http.ResponseWriter, body io.ReadCloser) {
	flusher, _ := w.(http.Flusher)
	buffer := make([]byte, 32*1024)
	for {
		n, err := readStreamBody(body, buffer, DefaultStreamIdleTimeout)
		if n > 0 {
			_, _ = w.Write(buffer[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
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
