package httputil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadStreamBodyTimesOutWhenIdle(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	buffer := make([]byte, 8)
	_, err := readStreamBody(reader, buffer, time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyResponseBodyPreservesTransferFailures(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			body    io.Reader
			length  string
			want    int64
			wantErr error
		}{
			{name: "complete", body: strings.NewReader("abc"), length: "3", want: 3},
			{name: "unknown", body: strings.NewReader("abc"), want: 3},
			{name: "truncated", body: strings.NewReader("abc"), length: "5", want: 3, wantErr: io.ErrUnexpectedEOF},
			{name: "excess", body: strings.NewReader("abc"), length: "2", want: 3, wantErr: io.ErrUnexpectedEOF},
			{name: "read error", body: &failingBodyReader{}, want: 3, wantErr: io.ErrUnexpectedEOF},
		} {
			t.Run(fmt.Sprintf("stream=%v/%s", streaming, tc.name), func(t *testing.T) {
				response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(tc.body), ContentLength: -1}
				if tc.length != "" {
					response.Header.Set("Content-Length", tc.length)
				}
				if streaming {
					response.Header.Set("Content-Type", "text/event-stream")
				}
				n, err := CopyResponseBody(httptest.NewRecorder(), response)
				if n != tc.want || !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %d, %v; want %d, %v", n, err, tc.want, tc.wantErr)
				}
			})
		}
	}
}

type failingBodyReader struct{}

func (*failingBodyReader) Read(p []byte) (int, error) { return copy(p, "abc"), io.ErrUnexpectedEOF }

type failingBodyWriter struct {
	*httptest.ResponseRecorder
	short bool
	flush bool
}

func (w *failingBodyWriter) Write(p []byte) (int, error) {
	if w.flush {
		return w.ResponseRecorder.Write(p)
	}
	if w.short {
		return 1, nil
	}
	return 1, io.ErrClosedPipe
}
func (w *failingBodyWriter) FlushError() error { return io.ErrClosedPipe }

func TestCopyResponseBodyReportsWriteFailures(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		for _, short := range []bool{false, true} {
			response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("abc"))}
			if streaming {
				response.Header.Set("Content-Type", "text/event-stream")
			}
			n, err := CopyResponseBody(&failingBodyWriter{ResponseRecorder: httptest.NewRecorder(), short: short}, response)
			want := io.ErrClosedPipe
			if short {
				want = io.ErrShortWrite
			}
			if n != 1 || !errors.Is(err, want) {
				t.Fatalf("stream=%v short=%v: %d, %v", streaming, short, n, err)
			}
		}
	}
	response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("abc"))}
	n, err := CopyResponseBody(&failingBodyWriter{ResponseRecorder: httptest.NewRecorder(), flush: true}, response)
	if n != 3 || !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("flush: %d, %v", n, err)
	}
}

func TestCopyResponseBodySkipsBodylessResponses(t *testing.T) {
	for _, tc := range []struct {
		method string
		status int
	}{{"HEAD", 200}, {"GET", 204}, {"GET", 304}} {
		response := &http.Response{StatusCode: tc.status, Request: httptest.NewRequest(tc.method, "/", nil), Header: http.Header{"Content-Length": {"100"}}, Body: io.NopCloser(&failingBodyReader{})}
		n, err := CopyResponseBody(httptest.NewRecorder(), response)
		if n != 0 || err != nil {
			t.Fatalf("%v: %d, %v", tc, n, err)
		}
	}
}

func TestReadResponseBodyRejectsLengthMismatch(t *testing.T) {
	response := &http.Response{Header: http.Header{"Content-Length": {"5"}}, Body: io.NopCloser(strings.NewReader("abc"))}
	if _, err := ReadResponseBody(response); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v", err)
	}
}

func (w *failingBodyWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestCopyResponseBodyPreservesHTTP2FailureDirection(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, upstream := range []bool{false, true} {
			t.Run(fmt.Sprintf("stream=%v/upstream=%v", stream, upstream), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				cancel()
				failure := errors.New("http2: stream closed")
				response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}
				if stream {
					response.Header.Set("Content-Type", "text/event-stream")
				}
				if upstream {
					response.Body = io.NopCloser(proxyFailingReader{err: failure})
				}
				writer := &proxyFailingWriter{ResponseRecorder: httptest.NewRecorder(), write: func([]byte) (int, error) { return 0, failure }}
				_, err := CopyResponseBody(writer, response)
				if !errors.Is(err, failure) || IsCanceledResponseTransfer(ctx, err) != !upstream {
					t.Fatalf("failure direction lost: %v", err)
				}
			})
		}
	}
}
