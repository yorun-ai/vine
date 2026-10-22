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
	"testing/synctest"
	"time"
)

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

func TestSSEIdleDoesNotHideTruncation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader, writer := io.Pipe()
		defer writer.Close()
		response := &http.Response{StatusCode: 200, Header: http.Header{
			"Content-Type": {"text/event-stream"}, "Content-Length": {"10"},
		}, Body: reader}
		_, err := CopyResponseBodyWithIdleTimeout(httptest.NewRecorder(), response, time.Second)
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("fixed-length idle response must remain truncated: %v", err)
		}
	})
}

func TestSSEIdleCloseAcrossHTTPProxies(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%v", h2), func(t *testing.T) {
			// Only the external gateway times out; internal forwarding has no idle limit.
			results := make(chan error, 2)
			closed := make(chan struct{})
			inner := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: ready\n\n"))
				_ = http.NewResponseController(w).Flush()
				<-r.Context().Done()
				close(closed)
			}))
			inner.EnableHTTP2 = h2
			inner.StartTLS()
			defer inner.Close()
			previous := inner
			for hop := range 2 {
				upstream := previous
				proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					request, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream.URL, nil)
					response, err := upstream.Client().Do(request)
					if err != nil {
						results <- err
						http.Error(w, err.Error(), 502)
						return
					}
					defer response.Body.Close()
					if (response.ProtoMajor == 2) != h2 {
						results <- fmt.Errorf("wrong upstream protocol: %s", response.Proto)
					}
					CopyHeader(w.Header(), response.Header)
					if hop == 0 {
						_, err = CopyResponseBody(w, response)
					} else {
						_, err = CopyResponseBodyWithIdleTimeout(w, response, 50*time.Millisecond)
					}
					// Portal closes its upstream body when idle; internal cancellation stays Debug.
					if IsCanceledResponseTransfer(r.Context(), err) {
						results <- nil
					} else {
						results <- err
					}
					if err != nil && !errors.Is(err, ErrStreamIdleTimeout) {
						panic(http.ErrAbortHandler)
					}
				}))
				proxy.EnableHTTP2 = h2
				proxy.StartTLS()
				defer proxy.Close()
				previous = proxy
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, previous.URL, nil)
			response, err := previous.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || string(body) != "data: ready\n\n" {
				t.Fatalf("response: %q, %v", body, err)
			}
			if (response.ProtoMajor == 2) != h2 {
				t.Fatalf("wrong client protocol: %s", response.Proto)
			}
			select {
			case <-closed:
			case <-ctx.Done():
				t.Fatal("upstream was not closed")
			}
			idle := 0
			for range 2 {
				if err := <-results; errors.Is(err, ErrStreamIdleTimeout) {
					idle++
				} else if err != nil {
					t.Fatalf("downstream failure: %v", err)
				}
			}
			if idle != 1 {
				t.Fatalf("idle closures: %d", idle)
			}
		})
	}
}

func TestInternalSSEForwardingHasNoIdleTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader, writer := io.Pipe()
		defer reader.Close()
		defer writer.Close()
		done := make(chan error, 1)
		recorder := httptest.NewRecorder()
		go func() {
			_, err := CopyResponseBody(recorder, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: reader, ContentLength: -1})
			done <- err
		}()
		synctest.Sleep(2 * DefaultStreamIdleTimeout)
		select {
		case err := <-done:
			t.Fatalf("internal idle closed stream: %v", err)
		default:
		}
		_, err := writer.Write([]byte("data: still alive\n\n"))
		if err != nil {
			t.Fatal(err)
		}
		writer.Close()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if !recorder.Flushed || recorder.Body.String() != "data: still alive\n\n" {
			t.Fatal("stream was not forwarded and flushed")
		}
	})
}

func TestSSEIdleDoesNotHideRequestTermination(t *testing.T) {
	for _, tc := range []struct {
		name       string
		newContext func(context.Context) (context.Context, context.CancelFunc)
		want       error
	}{
		{"deadline", func(parent context.Context) (context.Context, context.CancelFunc) {
			return context.WithTimeout(parent, time.Second)
		}, context.DeadlineExceeded},
		{"cancel", func(parent context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(parent)
			cancel()
			return ctx, cancel
		}, context.Canceled},
		{"idle cleanup", context.WithCancel, ErrStreamIdleTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := tc.newContext(t.Context())
				defer cancel()
				reader, writer := io.Pipe()
				defer writer.Close()
				body := &cancelOnCloseBody{ReadCloser: reader, cancel: cancel}
				response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body, ContentLength: -1, Request: httptest.NewRequestWithContext(ctx, "GET", "/", nil)}
				_, err := CopyResponseBodyWithIdleTimeout(httptest.NewRecorder(), response, time.Second)
				if !errors.Is(err, tc.want) {
					t.Fatalf("got %v, want %v", err, tc.want)
				}
				if ctx.Err() == nil {
					t.Fatal("body was not closed")
				}
			})
		})
	}
}

type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	b.cancel()
	return b.ReadCloser.Close()
}

func TestIdleStreamReaderReusesConnectionAcrossHeartbeats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader, writer := io.Pipe()
		defer writer.Close()
		response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: reader, ContentLength: -1}
		recorder := httptest.NewRecorder()
		done := make(chan error, 1)
		go func() {
			_, err := CopyResponseBodyWithIdleTimeout(recorder, response, time.Minute)
			done <- err
		}()
		for range 4 {
			synctest.Sleep(30 * time.Second)
			if _, err := writer.Write([]byte(": heartbeat\n\n")); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
		}
		synctest.Sleep(59 * time.Second)
		select {
		case err := <-done:
			t.Fatalf("stream closed early: %v", err)
		default:
		}
		synctest.Sleep(time.Second)
		if err := <-done; !errors.Is(err, ErrStreamIdleTimeout) {
			t.Fatalf("idle result: %v", err)
		}
		if recorder.Body.String() != strings.Repeat(": heartbeat\n\n", 4) {
			t.Fatal("heartbeat data changed")
		}
		if _, err := writer.Write([]byte("after close")); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("upstream not closed: %v", err)
		}
	})
}

func TestIdleStreamReaderCleansUpEveryExit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source io.Reader
		writer http.ResponseWriter
		want   error
	}{
		{"EOF", strings.NewReader("abc"), httptest.NewRecorder(), nil},
		{"read failure", &failingBodyReader{}, httptest.NewRecorder(), io.ErrUnexpectedEOF},
		{"write failure", strings.NewReader("abc"), &failingBodyWriter{ResponseRecorder: httptest.NewRecorder()}, io.ErrClosedPipe},
		{"short write", strings.NewReader("abc"), &failingBodyWriter{ResponseRecorder: httptest.NewRecorder(), short: true}, io.ErrShortWrite},
		{"flush failure", strings.NewReader("abc"), &failingBodyWriter{ResponseRecorder: httptest.NewRecorder(), flush: true}, io.ErrClosedPipe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// synctest also fails if the reader worker survives the copy's cleanup.
			synctest.Test(t, func(t *testing.T) {
				closed := false
				body := &cancelOnCloseBody{ReadCloser: io.NopCloser(tc.source), cancel: func() { closed = true }}
				response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body, ContentLength: -1}
				_, err := CopyResponseBodyWithIdleTimeout(tc.writer, response, time.Second)
				if !errors.Is(err, tc.want) {
					t.Fatalf("got %v, want %v", err, tc.want)
				}
				if !closed {
					t.Fatal("upstream body not closed")
				}
			})
		})
	}
}
